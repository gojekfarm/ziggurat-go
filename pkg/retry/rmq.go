package retry

import (
	"context"
	"github.com/gojekfarm/ziggurat-go/pkg/basic"
	"github.com/gojekfarm/ziggurat-go/pkg/logger"
	"github.com/gojekfarm/ziggurat-go/pkg/z"
	"github.com/makasim/amqpextra"
	"github.com/makasim/amqpextra/publisher"
	"github.com/streadway/amqp"
	"strings"
)

type RabbitMQConfig struct {
	Hosts                string
	DelayQueueExpiration string
}

func parseRabbitMQConfig(config z.ConfigReader) *RabbitMQConfig {
	rmqcfg := &RabbitMQConfig{}
	if err := config.UnmarshalByKey("rabbitmq", rmqcfg); err != nil {
		logger.LogError(err, "rmq config unmarshall error", nil)
		return &RabbitMQConfig{
			Hosts:                "amqp://user:bitnami@localhost:5672/",
			DelayQueueExpiration: "2000",
		}
	}
	return rmqcfg
}

func splitHosts(hosts string) []string {
	return strings.Split(hosts, ",")
}

type RabbitMQRetry struct {
	pdialer *amqpextra.Dialer
	cdialer *amqpextra.Dialer
	cfg     *RabbitMQConfig
}

func NewRabbitMQRetry(config z.ConfigReader) *RabbitMQRetry {
	cfg := parseRabbitMQConfig(config)
	return &RabbitMQRetry{
		cfg: cfg,
	}
}

func withChannel(connection *amqp.Connection, cb func(c *amqp.Channel) error) error {
	c, err := connection.Channel()
	defer c.Close()
	if err != nil {
		return err
	}
	cberr := cb(c)
	return cberr
}

func createDialer(ctx context.Context, hosts []string) (*amqpextra.Dialer, error) {
	d, cfgErr := amqpextra.NewDialer(
		amqpextra.WithURL(hosts...),
		amqpextra.WithContext(ctx))
	if cfgErr != nil {
		return nil, cfgErr
	}
	return d, nil
}

func (R *RabbitMQRetry) Start(app z.App) error {
	publishDialer, err := createDialer(app.Context(), splitHosts(R.cfg.Hosts))
	if err != nil {
		return err
	}
	R.pdialer = publishDialer
	conn, err := publishDialer.Connection(app.Context())
	if err != nil {
		return err
	}

	consumerDialer, err := createDialer(app.Context(), splitHosts(R.cfg.Hosts))
	if err != nil {
		return err
	}
	R.cdialer = consumerDialer

	if err := setupConsumers(app, consumerDialer); err != nil {
		return err
	}

	return withChannel(conn, func(c *amqp.Channel) error {
		createAndBindQueues(c, app.Router().GetTopicEntityNames(), app.Config().ServiceName)
		return nil
	})
}

func (R *RabbitMQRetry) Retry(app z.App, payload basic.MessageEvent) error {
	options := []publisher.Option{publisher.WithContext(app.Context())}
	p, err := R.pdialer.Publisher(options...)
	if err != nil {
		return err
	}
	defer p.Close()
	return retry(app.Context(), p, app.Config(), payload, "1000")
}

func (R *RabbitMQRetry) Stop() error {
	if R.pdialer != nil {
		R.pdialer.Close()
	}

	if R.cdialer != nil {
		R.cdialer.Close()
	}
	return nil
}

func (R *RabbitMQRetry) Replay(app z.App, topicEntity string, count int) error {
	return nil
}