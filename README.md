# opentelemetry-gitlabreceiver

**THIS IS WORK IN PROGRESS AND NOT READY TO BE USED!**

The Gitlabreciever creates OpenTelemetry traces for Gitlab pipelines. It leverages the Gitlab builtin webhooks. Each time a pipeline status changes, Gitlab emits a webhook to the Gitlabreceiver. The Gitlabreceiver transforms the received event into a trace. 

This project is **not** officially affiliated with the CNCF project [OpenTelemetry](https://opentelemetry.io/).

## Config

```yaml
receivers:
  gitlab: 
    endpoint: localhost:9286
    traces:
      url_path: "/v0.1/traces"
      refs: ["main", "master"] #By default all refs will be accpeted
service:
  pipelines:
    traces:
      receivers: [gitlab]
```

## Gitlab <-> Otel Mapping

Root Span = Pipeline \
Child Spans = Jobs 

### Trace creation 

If the Gitlab webhook event indicates that the pipeline is finished, the receiver will create a trace for the pipeline and all jobs within the pipleine. If a job within the pipleine is retried the reciever will create a **NEW** trace. 

-> Without any custom handling of the Gitlab events it is not possible (or at least I didn't find a way) to determine a suiting root-span-id without duplicating certain spans. 

If the Gitlab webhook is enabled for pipeline events it sends it for every status change. Usually that would be:

1. Pipeline creation 

    - Note: Default inital state appears to be "pending" - I was not able to get any other state right after pipeline creation.

2. Pipeline start 

    - Note: Default state would be running or created (if all jobs are manual)

3. Pipeline Finished 

    - Note: If all jobs are finished the pipeline will have a "finishedAt" time. This time will be used to determine the span end.

-> The Gitlabreceiver creates the trace for webhook event 3. Webhooks 1&2 are ignored for now.

### Usage 

To use the Gitlabreceiver a custom OpenTelemetry collector distribution needs to be created. This can be achieved with using the otel builder package and the following config. 

builder.yaml
```yaml 
dist:
  name: otelcol-dev
  description: Basic OTel Collector distribution for Developers
  output_path: ./otelcol-dev
  otelcol_version: 0.112.0

exporters:
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.112.0
  - gomod: go.opentelemetry.io/collector/exporter/otlpexporter v0.112.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.112.0

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.112.0
  - gomod: github.com/nw0rn/gitlabreceiver v0.101.0

providers:
  - gomod: go.opentelemetry.io/collector/confmap/provider/envprovider v1.17.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/fileprovider v1.17.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/httpprovider v1.17.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/httpsprovider v1.17.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/yamlprovider v1.17.0
```

```sh
go install go.opentelemetry.io/collector/cmd/builder@v0.112.0
builder --config=builder.yaml
```

