package ziggurat

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/rs/zerolog/log"
	"strings"
	"sync"
)

type topicEntity struct {
	handlerFunc      HandlerFunc
	consumers        []*kafka.Consumer
	bootstrapServers string
	originTopics     []string
	entityName       string
}

type TopicEntityHandlerMap = map[string]*topicEntity

type StreamRouter struct {
	handlerFunctionMap TopicEntityHandlerMap
}

func newConsumerConfig() *kafka.ConfigMap {
	return &kafka.ConfigMap{
		"bootstrap.servers":        "localhost:9092",
		"group.id":                 "myGroup",
		"auto.offset.reset":        "earliest",
		"enable.auto.commit":       true,
		"auto.commit.interval.ms":  2000,
		"debug":                    "consumer,broker",
		"enable.auto.offset.store": false,
	}
}

func NewRouter() *StreamRouter {
	return &StreamRouter{
		handlerFunctionMap: make(map[string]*topicEntity),
	}
}

func (sr *StreamRouter) GetHandlerFunctionMap() map[string]*topicEntity {
	return sr.handlerFunctionMap
}

func (sr *StreamRouter) GetTopicEntities() []*topicEntity {
	var topicEntities []*topicEntity
	for _, te := range sr.handlerFunctionMap {
		topicEntities = append(topicEntities, te)
	}
	return topicEntities
}

func (sr *StreamRouter) HandlerFunc(topicEntityName string, handlerFn HandlerFunc) {
	sr.handlerFunctionMap[topicEntityName] = &topicEntity{handlerFunc: handlerFn, entityName: topicEntityName}
}

func makeKV(key string, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func (sr *StreamRouter) stop() {
	for _, te := range sr.GetTopicEntities() {
		log.Info().Str("topic-entity", te.entityName).Msg("stopping consumers")
		for i, _ := range te.consumers {
			if closeErr := te.consumers[i].Close(); closeErr != nil {
				routerLogger.Error().Err(closeErr)
			}
		}
	}
}

func (sr *StreamRouter) Start(app *App) (chan int, error) {
	ctx := app.Context()
	config := app.Config
	stopChan := make(chan int)
	var wg sync.WaitGroup
	srConfig := config.StreamRouter
	hfMap := sr.handlerFunctionMap
	if len(hfMap) == 0 {
		routerLogger.Fatal().Err(ErrNoHandlersRegistered).Msg("")
	}

	for topicEntityName, te := range hfMap {
		streamRouterCfg := srConfig[topicEntityName]
		if topicEntityName != streamRouterCfg.TopicEntity {
			routerLogger.Fatal().Err(ErrTopicEntityMismatch).Msg("")
		}
		consumerConfig := newConsumerConfig()
		bootstrapServers := makeKV("bootstrap.servers", streamRouterCfg.BootstrapServers)
		groupID := makeKV("group.id", streamRouterCfg.GroupID)
		if setErr := consumerConfig.Set(bootstrapServers); setErr != nil {
			routerLogger.Error().Err(setErr).Msg("")
			return nil, setErr
		}
		if setErr := consumerConfig.Set(groupID); setErr != nil {
			routerLogger.Error().Err(setErr).Msg("")
			return nil, setErr
		}
		topics := strings.Split(streamRouterCfg.OriginTopics, ",")
		consumers := startConsumers(ctx, app, consumerConfig, topicEntityName, topics, streamRouterCfg.InstanceCount, te.handlerFunc, &wg)
		te.consumers = consumers
	}

	go func() {
		wg.Wait()
		sr.stop()
		close(stopChan)
	}()

	return stopChan, nil
}
