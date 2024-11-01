package gitlabreceiver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	conventions "go.opentelemetry.io/collector/semconv/v1.9.0"

	"go.opentelemetry.io/collector/pdata/pcommon"
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

	//We generate the trace and root span id based on a hash consisting out of several (unique) values
	traceId, err := getTraceId(p.Pipeline.Sha, strconv.Itoa(p.Pipeline.Id), p.Pipeline.FinishedAt)
	if err != nil {
		return nil, err
	}
	rootSpanId, err = getRootSpanId(p.Pipeline.Sha, strconv.Itoa(p.Pipeline.Id), p.Pipeline.FinishedAt)
	if err != nil {
		return nil, err
	}

	pipelineName := fmt.Sprintf("Gitlab Pipeline: %s - %s", strconv.Itoa(p.Pipeline.Id), p.Pipeline.Url)
	startTime, err := parseGitlabTime(p.Pipeline.CreatedAt)
	if err != nil {
		return nil, err
	}
	endTime, err := parseGitlabTime(p.Pipeline.FinishedAt)
	if err != nil {
		return nil, err
	}

	trace := ptrace.NewTraces()
	rss := trace.ResourceSpans()

	//Capacity is job count + pipeline + 1 (buffer)
	rss.EnsureCapacity(len(p.Jobs) + 1 + 1)
	rs := rss.AppendEmpty()
	rs.Resource().Attributes().PutStr(conventions.AttributeServiceName, p.Project.Path)
	rs.Resource().Attributes().PutStr(conventionsAttributeSpanSource, fmt.Sprintf("%s-receiver", typeStr.String()))

	//The pipeline span is the root span, therefore 0 bytes for the parentSpanId
	createSpan(rs, traceId, rootSpanId, [8]byte{0, 0, 0, 0, 0, 0, 0, 0}, pipelineName, startTime, endTime, p)

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
			createSpan(rs, traceId, getRandomSpanId(), rootSpanId, jobName, startedAt, finishedAt, j)
		}
	}
	return &trace, nil
}

// CICD Pipeline semconv: https://opentelemetry.io/docs/specs/semconv/attributes-registry/cicd/#cicd-pipeline-attributes
func (p glPipelineEvent) setAttributes(s ptrace.Span) {
	vc := len(p.Pipeline.Variables)
	s.Attributes().EnsureCapacity(12 + vc)
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineUrl, p.Pipeline.Url)
	s.Attributes().PutStr(conventionsAttributeCidCPipelineRunId, strconv.Itoa(p.Pipeline.Id))
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineDuration, strconv.Itoa(p.Pipeline.Duration))
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineQueuedDuration, strconv.Itoa(p.Pipeline.QueuedDuration))
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineUser, p.User.Name)
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineUsername, p.User.Username)
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineUserEmail, p.User.Email)

	s.Attributes().PutStr(conventionsAttributeCiCdPipelineCommitMessage, p.Commit.Message)
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineCommitTitle, p.Commit.Title)
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineCommitTimestamp, p.Commit.Timestamp)
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineCommitUrl, p.Commit.URL)
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineCommitAuthorEmail, p.Commit.Author.Email)

	for _, v := range p.Pipeline.Variables {
		s.Attributes().PutStr(fmt.Sprintf("%s.%s", conventionsAttributeCiCdPipelineVariable, v.Key), v.Value)
	}

	if p.Pipeline.Source == "parent_pipeline" {
		s.Attributes().PutStr(conventionsAttributeCiCdParentPipelineId, strconv.Itoa(p.ParentPipeline.Id))
		parentPipelineUrl := fmt.Sprintf("%s/pipelines/%s", p.ParentPipeline.Project.Url, strconv.Itoa(p.ParentPipeline.Id))
		s.Attributes().PutStr(conventionsAttributeCiCdParentPipelineUrl, parentPipelineUrl)
	}

	setSpanStatus(s, p.Pipeline.Status)
}

func (j Job) setAttributes(s ptrace.Span) {
	stage := strings.ToLower(j.Stage)
	switch {
	case strings.Contains(stage, "build"):
		stage = "build"
	case strings.Contains(stage, "test"):
		stage = "test"
	case strings.Contains(stage, "deploy"):
		stage = "deploy"
	}

	rtc := len(j.Runner.Tags)
	s.Attributes().EnsureCapacity(11 + rtc)
	s.Attributes().PutStr(conventionsAttributeCiCdTaskRunId, strconv.Itoa(j.Id))
	s.Attributes().PutStr(conventionsAttributeCiCdTaskRunUrl, j.Url)
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineTaskType, stage)
	s.Attributes().PutStr(conventionsAttributeCiCdJobEnvironment, j.Environment.Name)
	s.Attributes().PutStr(conventionsAttributeCiCdJobRunnerId, strconv.Itoa(j.Runner.Id))
	s.Attributes().PutStr(conventionsAttributeCiCdJobRunnerDescription, j.Runner.Description)
	s.Attributes().PutStr(conventionsAttributeCiCdJobRunnerIsActive, strconv.FormatBool(j.Runner.IsActive))
	s.Attributes().PutStr(conventionsAttributeCiCdJobRunnerIsShared, strconv.FormatBool(j.Runner.IsShared))
	s.Attributes().PutStr(conventionsAttributeCiCdJobDuration, strconv.Itoa(int(j.Duration)))
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineCommitMessage, j.Commit.Message)
	s.Attributes().PutStr(conventionsAttributeCiCdPipelineCommitTitle, j.Commit.Title)

	for _, t := range j.Runner.Tags {
		s.Attributes().PutStr(conventionsAttributeCiCdJobRunnerTag, t)
	}

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

func parseGitlabTime(t string) (pcommon.Timestamp, error) {
	if t == "" || t == "null" {
		return 0, nil
	}

	//For some reason the Gitlab test pipeline event has a different time format which we need to support to test (and eventually reenable webhooks) therefore we are continuing on error to handle the webhook test and the actual webhook
	pt, err := time.Parse(gitlabEventTimeFormat, t)
	if err == nil {
		return pcommon.NewTimestampFromTime(pt), nil
	}

	pt, err = time.Parse(time.RFC3339, t) //Time format of test pipeline events
	if err == nil {
		return pcommon.NewTimestampFromTime(pt), nil
	}

	return 0, err
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	err := json.NewDecoder(r.Body).Decode(&v)
	if err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}
