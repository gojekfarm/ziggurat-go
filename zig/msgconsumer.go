package zig

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"sync"
	"time"
)

var consumerLogContext = map[string]interface{}{"component": "consumer"}

const DefaultPollTimeout = 100 * time.Millisecond

func createConsumer(consumerConfig *kafka.ConfigMap, topics []string) *kafka.Consumer {
	consumer, err := kafka.NewConsumer(consumerConfig)
	logError(err, "ziggurat consumer", consumerLogContext)
	subscribeErr := consumer.SubscribeTopics(topics, nil)
	logError(subscribeErr, "ziggurat consumer", consumerLogContext)
	return consumer
}

func storeOffsets(consumer *kafka.Consumer, partition kafka.TopicPartition) error { // at least once delivery
	if partition.Error != nil {
		return ErrOffsetCommit
	}
	offsets := []kafka.TopicPartition{partition}
	offsets[0].Offset++
	if _, err := consumer.StoreOffsets(offsets); err != nil {
		return err
	}
	return nil
}

func startConsumer(routerCtx context.Context, app App, handlerFunc HandlerFunc, consumer *kafka.Consumer, topicEntity string, instanceID string, wg *sync.WaitGroup) {
	logInfo("ziggurat consumer: starting consumer", map[string]interface{}{"consumer-instance-id": instanceID})
	go func(routerCtx context.Context, c *kafka.Consumer, instanceID string, waitGroup *sync.WaitGroup) {
		doneCh := routerCtx.Done()
		for {
			select {
			case <-doneCh:
				wg.Done()
				return
			default:
				msg, err := c.ReadMessage(DefaultPollTimeout)
				if err != nil && err.(kafka.Error).Code() != kafka.ErrTimedOut {
				} else if err != nil && err.(kafka.Error).Code() == kafka.ErrAllBrokersDown {
					logError(err, "ziggurat consumer", nil)
					wg.Done()
					return
				}
				if msg != nil {
					messageEvent := MessageEvent{
						MessageKey:        nil,
						MessageValue:      nil,
						MessageValueBytes: msg.Value,
						MessageKeyBytes:   msg.Key,
						Topic:             *msg.TopicPartition.Topic,
						TopicEntity:       topicEntity,
						TimestampType:     msg.TimestampType.String(),
						KafkaTimestamp:    msg.Timestamp,
						Attributes:        make(map[string]interface{}),
					}
					MessageHandler(app, handlerFunc)(messageEvent)
					logError(storeOffsets(consumer, msg.TopicPartition), "ziggurat consumer", nil)
				}
			}
		}
	}(routerCtx, consumer, instanceID, wg)
}

var startConsumers = func(routerCtx context.Context, app App, consumerConfig *kafka.ConfigMap, topicEntity string, topics []string, instances int, handlerFunc HandlerFunc, wg *sync.WaitGroup) []*kafka.Consumer {
	consumers := make([]*kafka.Consumer, 0, instances)
	for i := 0; i < instances; i++ {
		consumer := createConsumer(consumerConfig, topics)
		consumers = append(consumers, consumer)
		groupID, _ := consumerConfig.Get("group.id", "")
		instanceID := fmt.Sprintf("%s-%s-%d", topicEntity, groupID, i)
		wg.Add(1)
		startConsumer(routerCtx, app, handlerFunc, consumer, topicEntity, instanceID, wg)
	}
	return consumers
}
