package retry

import (
	"context"
	"errors"
	"github.com/gojekfarm/ziggurat-go/pkg/basic"
	"github.com/gojekfarm/ziggurat-go/pkg/z"
	"github.com/makasim/amqpextra"
	"github.com/streadway/amqp"
	"reflect"
	"testing"
	"time"
)

const entityName = "foo"

type retryMockConfigReader struct{}
type retryMockApp struct{}
type retryMockRouter struct{}

func (r retryMockRouter) Start(app z.App) (chan struct{}, error) {
	panic("implement me")
}

func (r retryMockRouter) HandlerFunc(topicEntityName string, handlerFn z.HandlerFunc, mw ...z.MiddlewareFunc) {
	panic("implement me")
}

func (r retryMockRouter) GetTopicEntities() []*z.TopicEntity {
	return []*z.TopicEntity{{
		HandlerFunc: func(messageEvent basic.MessageEvent, app z.App) z.ProcessStatus {
			return z.ProcessingSuccess
		},
		Consumers:  nil,
		EntityName: entityName,
	}}
}

func (r retryMockRouter) GetHandlerFunctionMap() map[string]*z.TopicEntity {
	panic("implement me")
}

func (r retryMockRouter) GetTopicEntityNames() []string {
	return []string{entityName}
}

func (r retryMockApp) Context() context.Context {
	return context.Background()
}

func (r retryMockApp) Router() z.StreamRouter {
	return &retryMockRouter{}
}

func (r retryMockApp) MessageRetry() z.MessageRetry {
	panic("implement me")
}

func (r retryMockApp) Run(router z.StreamRouter, options z.RunOptions) chan struct{} {
	panic("implement me")
}

func (r retryMockApp) Configure(configFunc func(o z.App) z.Options) {
	panic("implement me")
}

func (r retryMockApp) MetricPublisher() z.MetricPublisher {
	panic("implement me")
}

func (r retryMockApp) HTTPServer() z.HttpServer {
	panic("implement me")
}

func (r retryMockApp) Config() *basic.Config {
	return &basic.Config{
		ServiceName: "baz",
	}
}

func (r retryMockApp) ConfigReader() z.ConfigReader {
	panic("implement me")
}

func (r retryMockApp) Stop() {
	panic("implement me")
}

func (r retryMockApp) IsRunning() bool {
	panic("implement me")
}

func (r retryMockConfigReader) Config() *basic.Config {
	panic("implement me")
}

func (r retryMockConfigReader) Parse(options basic.CommandLineOptions) {
	panic("implement me")
}

func (r retryMockConfigReader) GetByKey(key string) interface{} {
	panic("implement me")
}

func (r retryMockConfigReader) Validate(rules map[string]func(c *basic.Config) error) error {
	panic("implement me")
}

func (r retryMockConfigReader) UnmarshalByKey(key string, model interface{}) error {
	return nil
}

func TestRabbitMQRetry_StartWithDialerError(t *testing.T) {
	retryMockConfigReader := &retryMockConfigReader{}
	app := &retryMockApp{}
	rmq := NewRabbitMQRetry(retryMockConfigReader)
	dialerError := errors.New("dialer error")
	oldCreateDialer := createDialer
	defer func() {
		createDialer = oldCreateDialer
	}()
	createDialer = func(ctx context.Context, hosts []string, dialTimeout time.Duration) (*amqpextra.Dialer, error) {
		return nil, dialerError
	}
	err := rmq.Start(app)
	if err != dialerError {
		t.Errorf("expected error to be %v got %v", dialerError, err)
	}
}

func TestRabbitMQRetry_StartSuccess(t *testing.T) {
	retryMockConfigReader := &retryMockConfigReader{}
	expectedServiceName := "baz"
	expectedEntities := []string{entityName}
	app := &retryMockApp{}
	rmq := NewRabbitMQRetry(retryMockConfigReader)
	createDialer = func(ctx context.Context, hosts []string, dialTimeout time.Duration) (*amqpextra.Dialer, error) {
		return &amqpextra.Dialer{}, nil
	}
	setupConsumers = func(app z.App, dialer *amqpextra.Dialer) error {
		return nil
	}
	getConnectionFromDialer = func(ctx context.Context, d *amqpextra.Dialer) (*amqp.Connection, error) {
		return &amqp.Connection{}, nil
	}
	withChannel = func(connection *amqp.Connection, cb func(c *amqp.Channel) error) error {
		cb(&amqp.Channel{})
		return nil
	}
	createAndBindQueues = func(c *amqp.Channel, topicEntities []string, serviceName string) {
		if serviceName != expectedServiceName {
			t.Errorf("expected servicename %s got %s", expectedServiceName, serviceName)
		}
		if !reflect.DeepEqual(topicEntities, expectedEntities) {
			t.Errorf("expected entities to be %v got %v", expectedEntities, topicEntities)
		}
	}
	err := rmq.Start(app)
	if err != nil {
		t.Errorf("expected error to nil")
	}

}