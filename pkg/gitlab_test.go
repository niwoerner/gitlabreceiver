package gitlabreceiver

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	gitlabPipelineEvent = `{
    "object_kind": "pipeline",
    "object_attributes": {
        "id": 1234567890, 
		"status": "pending"
    },
    "builds": [
        {
            "id": 7961245403,
            "name": "job1",
            "status": "pending"
        }
    ]
}`
	gitlabStartTime = "2024-01-01 12:30:15 UTC"
	gitlabEndTime   = "2024-01-01 12:40:15 UTC"
)

func TestCreateSpan(t *testing.T) {
	tests := []struct {
		commitSHA       string
		pipelineId      string
		name            string
		statusCode      ptrace.StatusCode
		traceId         [16]byte
		spanId          [8]byte
		parentId        [8]byte
		startTime       pcommon.Timestamp
		endTime         pcommon.Timestamp
		glPipelineEvent glPipelineEvent
	}{
		//root span - pipeline pending (=inital state when a pipeline is launched)
		{
			commitSHA:  "abc123",
			pipelineId: "123456",
			name:       "pipeline: 123456",
			traceId:    generateExpectedTraceId("abc123", "pipeline1", "10"),
			spanId:     generateExpectedSpanId("abc123", "321abc", "10"),
			parentId:   [8]byte{0, 0, 0, 0, 0, 0, 0, 0},
			startTime:  getParsedGitlabTime(gitlabStartTime),
			endTime:    getParsedGitlabTime("null"),
			statusCode: ptrace.StatusCodeError,
			glPipelineEvent: glPipelineEvent{
				Pipeline: Pipeline{
					Status: "failed",
				},
			},
		},
		//root span - pipeline finished
		{
			commitSHA:  "abc123",
			pipelineId: "123456",
			name:       "pipeline: 123456",
			traceId:    generateExpectedTraceId("abc123", "pipeline1", "10"),
			spanId:     generateExpectedSpanId("abc123", "321abc", "10"),
			parentId:   [8]byte{0, 0, 0, 0, 0, 0, 0, 0},
			startTime:  getParsedGitlabTime(gitlabStartTime),
			endTime:    getParsedGitlabTime(gitlabEndTime),
			statusCode: ptrace.StatusCodeError,
			glPipelineEvent: glPipelineEvent{
				Pipeline: Pipeline{
					Status: "failed",
				},
			},
		},
		//child span - job created
		{
			commitSHA:  "abc123",
			pipelineId: "123456",
			name:       "job: 123456",
			traceId:    generateExpectedTraceId("abc123", "pipeline1", "10"),
			spanId:     generateExpectedSpanId("def123", "def321", "10"),
			parentId:   generateExpectedSpanId("abc123", "321abc", "10"),
			startTime:  getParsedGitlabTime(gitlabStartTime),
			endTime:    getParsedGitlabTime("null"),
			statusCode: ptrace.StatusCodeOk,
			glPipelineEvent: glPipelineEvent{
				Pipeline: Pipeline{
					Status: "created",
				},
			},
		},
		//child span - job finished
		{
			commitSHA:  "abc123",
			pipelineId: "123456",
			name:       "job: 123456",
			traceId:    generateExpectedTraceId("abc123", "pipeline1", "10"),
			spanId:     generateExpectedSpanId("def123", "def321", "10"),
			parentId:   generateExpectedSpanId("abc123", "321abc", "10"),
			startTime:  getParsedGitlabTime(gitlabStartTime),
			endTime:    getParsedGitlabTime(gitlabEndTime),
			statusCode: ptrace.StatusCodeOk,
			glPipelineEvent: glPipelineEvent{
				Pipeline: Pipeline{
					Status: "created",
				},
			},
		},
	}

	rs := ptrace.NewResourceSpans()
	resourceScopeSpans := rs.ScopeSpans()
	for i, test := range tests {
		createSpan(rs, test.traceId, test.spanId, test.parentId, test.name, "not-used", test.startTime, test.endTime, &test.glPipelineEvent)
		testSpan := resourceScopeSpans.At(i).Spans().At(0)
		assert.Equal(t, hex.EncodeToString(test.traceId[:]), testSpan.TraceID().String(), "TraceID should be set correctly")
		assert.Equal(t, hex.EncodeToString(test.spanId[:]), testSpan.SpanID().String(), "SpanID should be set correctly")
		if test.parentId == [8]byte{0, 0, 0, 0, 0, 0, 0, 0} {
			assert.Equal(t, "", testSpan.ParentSpanID().String(), "ParentSpanID should not be set")
		} else {
			assert.Equal(t, hex.EncodeToString(test.parentId[:]), testSpan.ParentSpanID().String(), "ParentSpanID should be set correctly")
		}
		assert.Equal(t, test.name, testSpan.Name(), "Name should be set correctly")
		assert.Equal(t, test.statusCode, testSpan.Status().Code(), "Status should be set correctly")
		assert.Equal(t, test.startTime, testSpan.StartTimestamp(), "StartTimestamp should be set correctly")
		assert.Equal(t, test.endTime, testSpan.EndTimestamp(), "EndTimestamp should be set correctly")
	}
}

func getParsedGitlabTime(s string) pcommon.Timestamp {
	t, _ := parseGitlabTime(s)
	return t
}

func newGlPipelineEvent() *glPipelineEvent {
	return &glPipelineEvent{
		Kind: "pipeline",
		Pipeline: Pipeline{
			Id:         1480179747,
			Status:     "success",
			Duration:   120,
			Url:        "https://gitlab.com/project/pipeline/1480179747",
			CreatedAt:  time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			FinishedAt: time.Now().Format(time.RFC3339),
			Sha:        "abc123def456",
			Source:     "push",
		},
		Jobs: []Job{
			{
				Id:          101,
				Name:        "build",
				Status:      "success",
				Stage:       "build",
				CreatedAt:   time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
				StartedAt:   time.Now().Add(-9 * time.Minute).Format(time.RFC3339),
				FinishedAt:  time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
				Url:         "https://gitlab.com/project/-/jobs/101",
				ProjectPath: "group/project",
			},
			{
				Id:          102,
				Name:        "deploy",
				Status:      "success",
				Stage:       "deploy",
				CreatedAt:   time.Now().Add(-6 * time.Minute).Format(time.RFC3339),
				StartedAt:   time.Now().Add(-4 * time.Minute).Format(time.RFC3339),
				FinishedAt:  time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
				Url:         "https://gitlab.com/project/-/jobs/102",
				ProjectPath: "group/project",
			},
		},
		Project: Project{
			Name: "example-project",
			Id:   12345,
			Path: "group/example-project",
			Url:  "https://gitlab.com/group/example-project",
		},
		ParentPipeline: ParentPipeline{
			Id: 1,
			Project: Project{
				Name: "parent-project",
				Id:   54321,
				Path: "group/parent-project",
				Url:  "https://gitlab.com/group/parent-project",
			},
		},
	}
}
