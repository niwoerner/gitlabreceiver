package gitlabreceiver

import (
	"encoding/hex"
	"testing"

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

func PipelineEvent_SetAttributes(t *testing.T) {
	tests := []struct {
		name     string
		event    glPipelineEvent
		expected map[string]string
	}{
		{
			name: "With parent pipeline",
			event: glPipelineEvent{
				Pipeline: Pipeline{
					Url:    "https://gitlab.com/test-pipeline",
					Id:     123,
					Source: "parent_pipeline",
					Status: "success",
				},
				ParentPipeline: ParentPipeline{
					Id: 456,
					Project: Project{
						Url: "https://gitlab.com/test-parent-project",
					},
				},
			},
			expected: map[string]string{
				conventionsAttributeCiCdPipelineUrl:       "https://gitlab.com/test-pipeline",
				conventionsAttributeCidCPipelineRunId:     "123",
				conventionsAttributeCiCdParentPipelineId:  "456",
				conventionsAttributeCiCdParentPipelineUrl: "https://gitlab.com/test-parent-project/pipelines/456",
			},
		},
		{
			name: "Without parent pipeline",
			event: glPipelineEvent{
				Pipeline: Pipeline{
					Url:    "https://gitlab.com/test-pipeline",
					Id:     124,
					Source: "direct",
					Status: "failed",
				},
				ParentPipeline: ParentPipeline{}, // No parent
			},
			expected: map[string]string{
				conventionsAttributeCiCdPipelineUrl:   "https://gitlab.com/test-pipeline",
				conventionsAttributeCidCPipelineRunId: "124",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := ptrace.NewSpan()
			tt.event.setAttributes(span)

			for key, expectedValue := range tt.expected {
				if actualValue, exists := span.Attributes().Get(key); !exists || actualValue.Str() != expectedValue {
					t.Errorf("expected %s to be %s, got %s", key, expectedValue, actualValue.Str())
				}
			}
		})
	}
}

func Job_SetAttributes(t *testing.T) {
	tests := []struct {
		name     string
		job      Job
		expected map[string]string
	}{
		{
			name: "Successful job",
			job: Job{
				Id:     789,
				Url:    "https://gitlab.com/test-job-success",
				Stage:  "test",
				Status: "success",
			},
			expected: map[string]string{
				conventionsAttributeCiCdTaskRunId:        "789",
				conventionsAttributeCiCdTaskRunUrl:       "https://gitlab.com/test-job-success",
				conventionsAttributeCiCdPipelineTaskType: "test",
			},
		},
		{
			name: "Failed job",
			job: Job{
				Id:     790,
				Url:    "https://gitlab.com/test-job-fail",
				Stage:  "deploy",
				Status: "failed",
			},
			expected: map[string]string{
				conventionsAttributeCiCdTaskRunId:        "790",
				conventionsAttributeCiCdTaskRunUrl:       "https://gitlab.com/test-job-fail",
				conventionsAttributeCiCdPipelineTaskType: "deploy",
			},
		},
		{
			name: "Job with empty URL",
			job: Job{
				Id:     791,
				Url:    "",
				Stage:  "build",
				Status: "success",
			},
			expected: map[string]string{
				conventionsAttributeCiCdTaskRunId:        "791",
				conventionsAttributeCiCdTaskRunUrl:       "",
				conventionsAttributeCiCdPipelineTaskType: "build",
			},
		},
		{
			name: "Job with empty stage",
			job: Job{
				Id:     792,
				Url:    "https://gitlab.com/test-job-empty-stage",
				Stage:  "",
				Status: "success",
			},
			expected: map[string]string{
				conventionsAttributeCiCdTaskRunId:        "792",
				conventionsAttributeCiCdTaskRunUrl:       "https://gitlab.com/test-job-empty-stage",
				conventionsAttributeCiCdPipelineTaskType: "",
			},
		},
		{
			name: "Job with special characters in URL",
			job: Job{
				Id:     793,
				Url:    "https://gitlab.com/test-job-@#$%&",
				Stage:  "integration",
				Status: "success",
			},
			expected: map[string]string{
				conventionsAttributeCiCdTaskRunId:        "793",
				conventionsAttributeCiCdTaskRunUrl:       "https://gitlab.com/test-job-@#$%&",
				conventionsAttributeCiCdPipelineTaskType: "integration",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := ptrace.NewSpan()
			tt.job.setAttributes(span)

			for key, expectedValue := range tt.expected {
				if actualValue, exists := span.Attributes().Get(key); !exists || actualValue.Str() != expectedValue {
					t.Errorf("expected %s to be %s, got %s", key, expectedValue, actualValue.Str())
				}
			}
		})
	}
}

func TestSetSpanStatus(t *testing.T) {
	tests := []struct {
		name            string
		status          string
		expectedCode    ptrace.StatusCode
		expectedMessage string
	}{
		{
			name:            "Failed status",
			status:          "failed",
			expectedCode:    ptrace.StatusCodeError,
			expectedMessage: "failed",
		},
		{
			name:            "Successful status",
			status:          "success",
			expectedCode:    ptrace.StatusCodeOk,
			expectedMessage: "success",
		},
		//In progress and unknown is considered "ok" for now.
		{
			name:            "In progress status",
			status:          "in_progress",
			expectedCode:    ptrace.StatusCodeOk,
			expectedMessage: "in_progress",
		},
		{
			name:            "Unknown status",
			status:          "unknown",
			expectedCode:    ptrace.StatusCodeOk,
			expectedMessage: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := ptrace.NewSpan()
			setSpanStatus(span, tt.status)

			if span.Status().Code() != tt.expectedCode {
				t.Errorf("expected status code %v, got %v", tt.expectedCode, span.Status().Code())
			}
			if span.Status().Message() != tt.expectedMessage {
				t.Errorf("expected status message %s, got %s", tt.expectedMessage, span.Status().Message())
			}
		})
	}
}
