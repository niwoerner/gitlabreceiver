# opentelemetry-gitlabreceiver

**THIS IS WORK IN PROGRESS AND NOT READY TO BE USED!**

The Gitlabreciever creates OpenTelemetry traces for Gitlab pipelines. It leverages the Gitlab builtin webhooks. Each time a pipeline status changes, Gitlab emits a webhook to the Gitlabreceiver. The Gitlabreceiver transforms the received event into a trace. 

This project is **not** officially affiliated with the CNCF project [OpenTelemetry](https://opentelemetry.io/).

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