package gitlabreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	defaultInterval      = 10 * time.Second
	defaultTracesUrlPath = "/v0.1/traces"
)

var typeStr = component.MustNewType("gitlabreceiver")

type HTTPConfig struct {
	confighttp.ServerConfig `mapstructure:",squash"`

	TracesURLPath string `mapstructure:"traces_url_path,omitempty"`
}

type Protocols struct {
	HTTP *HTTPConfig `mapstructure:"http"`
}

type Config struct {
	Interval  string `mapstructure:"interval"`
	Protocols `mapstructure:"protocols"`
}

// ToDo: Define expected config struct and implement validation
func (cfg *Config) Validate() error {
	return nil
}

func createDefaultConfig() component.Config {
	return &Config{
		Interval: defaultInterval.String(),
		Protocols: Protocols{
			HTTP: &HTTPConfig{
				ServerConfig: confighttp.ServerConfig{
					Endpoint: "0.0.0.0:4318", //ToDo: Should not be 0.0.0.0
				},
				TracesURLPath: defaultTracesUrlPath,
			},
		},
	}
}

func createTracesReceiver(ctx context.Context, settings receiver.Settings, cfg component.Config, consumer consumer.Traces) (receiver.Traces, error) {
	glRcvr := newGitlabReceiver(cfg, settings)
	glRcvr.nextTracesConsumer = consumer

	return glRcvr, nil
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, component.StabilityLevelDevelopment))
}
