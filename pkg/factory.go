package gitlabreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

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
