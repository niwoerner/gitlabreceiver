package gitlabreceiver

const (
	gitlabEventTimeFormat = "2006-01-02 15:04:05 UTC" //iso8601Format

	//Semconv 1.27.0: https://opentelemetry.io/docs/specs/semconv/attributes-registry/cicd/#cicd-pipeline-attributes
	conventionsAttributeCiCdPipelineName     = "cicd.pipeline.name"
	conventionsAttributeCidCPipelineRunId    = "cicd.pipeline.run.id"
	conventionsAttributeCiCdPipelineTaskType = "cicd.pipeline.task.type" //In Gitlab a stage can be seen as task type -> well known values: build,deploy,test

	conventionsAttributeCiCdTaskRunId  = "cicd.pipeline.task.run.id"
	conventionsAttributeCiCdTaskRunUrl = "cicd.pipeline.task.run.url.full"

	//Custom Attributes - not part of Semconv 1.27.0

	//General
	conventionsAttributeSpanSource = "span.source"

	//Pipeline
	conventionsAttributeCiCdPipelineUrl            = "cicd.pipeline.url"
	conventionsAttributeCiCdParentPipelineId       = "cicd.parent.pipeline.run.id"
	conventionsAttributeCiCdParentPipelineUrl      = "cicd.parent.pipeline.url"
	conventionsAttributeCiCdPipelineDuration       = "cicd.pipeline.duration"
	conventionsAttributeCiCdPipelineQueuedDuration = "cicd.pipeline.queued.duration"
	conventionsAttributeCiCdPipelineVariable       = "cicd.pipeline.variable"
	conventionsAttributeCiCdPipelineUser           = "cicd.pipeline.user"
	conventionsAttributeCiCdPipelineUsername       = "cicd.pipeline.username"
	conventionsAttributeCiCdPipelineUserEmail      = "cicd.pipeline.user.email"

	conventionsAttributeCiCdPipelineCommitMessage     = "cicd.pipeline.commit.message"
	conventionsAttributeCiCdPipelineCommitTitle       = "cicd.pipeline.commit.title"
	conventionsAttributeCiCdPipelineCommitTimestamp   = "cicd.pipeline.commit.timestamp"
	conventionsAttributeCiCdPipelineCommitUrl         = "cicd.pipeline.commit.url"
	conventionsAttributeCiCdPipelineCommitAuthorEmail = "cicd.pipeline.commit.author.email"

	//Job
	conventionsAttributeCiCdJobEnvironment = "cicd.job.environment"

	conventionsAttributeCiCdJobRunnerId          = "cicd.job.runner.id"
	conventionsAttributeCiCdJobRunnerDescription = "cicd.job.runner.description"
	conventionsAttributeCiCdJobRunnerIsActive    = "cicd.job.runner.active"
	conventionsAttributeCiCdJobRunnerIsShared    = "cicd.job.runner.shared"
	conventionsAttributeCiCdJobRunnerTag         = "cicd.job.runner.tag"
)
