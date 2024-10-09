package gitlabreceiver

import (
	"fmt"
	"strconv"

	conventions "go.opentelemetry.io/collector/semconv/v1.9.0"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

type gitlabResource interface {
	newTrace() (*ptrace.Traces, error)
	setAttributes(ptrace.Span)
}

// The whole pipeline is the root span which defines the trace
func (p *glPipelineEvent) newTrace() (*ptrace.Traces, error) {
	var traceId [16]byte
	var rootSpanId [8]byte
	trace := ptrace.NewTraces()
	resourceSpansSlice := trace.ResourceSpans()
	rs := resourceSpansSlice.AppendEmpty()

	//Resource Attributes - will also be present on the job spans
	rs.Resource().Attributes().PutStr(conventions.AttributeServiceName, p.Project.Path)

	traceId, rootSpanId = generateTraceParentHeader(p.Pipeline.Sha, strconv.Itoa(p.Pipeline.Id), p.Pipeline.FinishedAt)

	pipelineName := fmt.Sprintf("Gitlab Pipeline: %s - %s", strconv.Itoa(p.Pipeline.Id), p.Pipeline.Url)
	startTime, err := parseGitlabTime(p.Pipeline.CreatedAt)
	if err != nil {
		return nil, err
	}

	endTime, err := parseGitlabTime(p.Pipeline.FinishedAt)
	if err != nil {
		return nil, err
	}

	//The pipeline span is the root span, therefore 0 bytes for the parentSpanId
	createSpan(rs, traceId, rootSpanId, [8]byte{0, 0, 0, 0, 0, 0, 0, 0}, pipelineName, p.Pipeline.Status, startTime, endTime, p)

	for _, j := range p.Jobs {
		if j.FinishedAt != "" {
			jobUrl := fmt.Sprintf("%s/jobs/%s", p.Project.Url, strconv.Itoa(j.Id))
			jobName := fmt.Sprintf("Job: %s - %s - Stage: %s", j.Name, strconv.Itoa(j.Id), j.Stage)
			j.setDetails(jobUrl)

			startedAt, err := parseGitlabTime(j.StartedAt)
			if err != nil {
				return nil, err
			}
			finishedAt, err := parseGitlabTime(j.FinishedAt)
			if err != nil {
				return nil, err
			}
			createSpan(rs, traceId, newSpanId(), rootSpanId, jobName, j.Status, startedAt, finishedAt, j)
		}
	}
	return &trace, nil
}

func (p glPipelineEvent) setAttributes(s ptrace.Span) {
	//CICD Pipeline semconv: https://opentelemetry.io/docs/specs/semconv/attributes-registry/cicd/#cicd-pipeline-attributes
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineUrl, p.Pipeline.Url)
	s.Attributes().PutStr(conventionsAttributeCidCPipelineRunId, strconv.Itoa(p.Pipeline.Id))
	if p.Pipeline.Source == "parent_pipeline" {
		s.Attributes().PutStr(conventionsAttributeCiCdParentPipelineId, strconv.Itoa(p.ParentPipeline.Id))
		parentPipelineUrl := fmt.Sprintf("%s/pipelines/%s", p.ParentPipeline.Project.Url, strconv.Itoa(p.ParentPipeline.Id))
		s.Attributes().PutStr(conventionsAttributeCiCdParentPipelineUrl, parentPipelineUrl)
	}
	setSpanStatus(s, p.Pipeline.Status)
}

func (j Job) setAttributes(s ptrace.Span) {
	// CICD Job semconv: https://opentelemetry.io/docs/specs/semconv/attributes-registry/cicd/#cicd-pipeline-attributes
	s.Attributes().PutStr(conventionsAttributeCiCdTaskRunId, strconv.Itoa(j.Id))
	s.Attributes().PutStr(conventionsAttributeCiCdTaskRunUrl, j.Url)
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineTaskType, j.Stage)

	setSpanStatus(s, j.Status)
}

func (j Job) newTrace() (*ptrace.Traces, error) {
	return nil, nil
}

func setSpanStatus(s ptrace.Span, status string) {
	if status == "failed" {
		s.Status().SetCode(ptrace.StatusCodeError)
	} else {
		s.Status().SetCode(ptrace.StatusCodeOk)
	}
	s.Status().SetMessage(status)
}

// Set additional job fields/details which are not getting captured automatically by deocding the gitlab event webhook
func (j *Job) setDetails(url string) {
	j.Url = url
}
