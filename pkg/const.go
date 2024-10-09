package gitlabreceiver

const (
	gitlabEventTimeFormat = "2006-01-02 15:04:05 UTC"
	supportedContentType  = "application/json"

	//Semconv 1.27.0: https://opentelemetry.io/docs/specs/semconv/attributes-registry/cicd/#cicd-pipeline-attributes
	conventionsAttributeCiCdPipelineName     = "cicd.pipeline.name"
	conventionsAttributeCidCPipelineRunId    = "cicd.pipeline.run.id"
	conventionsAttributeCiCdPipelineTaskType = "cicd.pipeline.task.type" //In Gitlab a stage can be seen as task type -> well known values: build,deploy,test

	conventionsAttributeCiCdTaskRunId  = "cicd.pipeline.task.run.id"
	conventionsAttributeCiCdTaskRunUrl = "cicd.pipeline.task.run.url.full"

	//Custom Attributes - not part of Semconv 1.27.0
	conventionsAttributeCiCdPipelineUrl       = "cicd.pipeline.url"
	conventionsAttributeCiCdParentPipelineId  = "cicd.parent.pipeline.run.id"
	conventionsAttributeCiCdParentPipelineUrl = "cicd.parent.pipeline.url"
)
