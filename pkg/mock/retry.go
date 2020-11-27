package mock

import (
	"github.com/gojekfarm/ziggurat-go/pkg/z"
	"github.com/gojekfarm/ziggurat-go/pkg/zbasic"
)

type Retry struct {
	StartFunc  func(app z.App) error
	RetryFunc  func(app z.App, payload zbasic.MessageEvent) error
	StopFunc   func() error
	ReplayFunc func(app z.App, topicEntity string, count int) error
}

func NewRetry() *Retry {
	return &Retry{
		StartFunc: func(app z.App) error {
			return nil
		},
		RetryFunc: func(app z.App, payload zbasic.MessageEvent) error {
			return nil
		},
		StopFunc: func() error {
			return nil
		},
		ReplayFunc: func(app z.App, topicEntity string, count int) error {
			return nil
		},
	}
}

func (m *Retry) Start(app z.App) error {
	return m.StartFunc(app)
}

func (m *Retry) Retry(app z.App, payload zbasic.MessageEvent) error {
	return m.Retry(app, payload)
}

func (m *Retry) Stop() error {
	return m.StopFunc()
}

func (m *Retry) Replay(app z.App, topicEntity string, count int) error {
	return m.ReplayFunc(app, topicEntity, count)
}
