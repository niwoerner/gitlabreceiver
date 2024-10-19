package gitlabreceiver

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		createSpan(rs, test.traceId, test.spanId, test.parentId, test.name, test.startTime, test.endTime, &test.glPipelineEvent)
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
			name: "With parent pipeline and variables",
			event: glPipelineEvent{
				Pipeline: Pipeline{
					Url:            "https://gitlab.com/test-pipeline",
					Id:             123,
					Source:         "parent_pipeline",
					Status:         "success",
					Duration:       3600,
					QueuedDuration: 120,
					Variables: []Variables{
						{Key: "ENV", Value: "production"},
						{Key: "DEBUG", Value: "false"},
					},
				},
				ParentPipeline: ParentPipeline{
					Id: 456,
					Project: Project{
						Url: "https://gitlab.com/test-parent-project",
					},
				},
				User: User{
					Name:     "John Doe",
					Username: "johndoe",
					Email:    "john@example.com",
				},
				Commit: Commit{
					Message:   "Fix pipeline issue",
					Title:     "Pipeline fix",
					Timestamp: "2024-10-19T12:00:00Z",
					URL:       "https://gitlab.com/commit/789",
					Author:    Author{Email: "author@example.com"},
				},
			},
			expected: map[string]string{
				conventionsAttributeCiCdPipelineUrl:               "https://gitlab.com/test-pipeline",
				conventionsAttributeCidCPipelineRunId:             "123",
				conventionsAttributeCiCdPipelineDuration:          "3600",
				conventionsAttributeCiCdPipelineQueuedDuration:    "120",
				conventionsAttributeCiCdPipelineUser:              "John Doe",
				conventionsAttributeCiCdPipelineUsername:          "johndoe",
				conventionsAttributeCiCdPipelineUserEmail:         "john@example.com",
				conventionsAttributeCiCdPipelineCommitMessage:     "Fix pipeline issue",
				conventionsAttributeCiCdPipelineCommitTitle:       "Pipeline fix",
				conventionsAttributeCiCdPipelineCommitTimestamp:   "2024-10-19T12:00:00Z",
				conventionsAttributeCiCdPipelineCommitUrl:         "https://gitlab.com/commit/789",
				conventionsAttributeCiCdPipelineCommitAuthorEmail: "author@example.com",
				conventionsAttributeCiCdParentPipelineId:          "456",
				conventionsAttributeCiCdParentPipelineUrl:         "https://gitlab.com/test-parent-project/pipelines/456",

				// Variable assertions
				fmt.Sprintf("%s.%s", conventionsAttributeCiCdPipelineVariable, "ENV"):   "production",
				fmt.Sprintf("%s.%s", conventionsAttributeCiCdPipelineVariable, "DEBUG"): "false",
			},
		},
		{
			name: "Without parent pipeline and commit info",
			event: glPipelineEvent{
				Pipeline: Pipeline{
					Url:            "https://gitlab.com/test-pipeline",
					Id:             124,
					Source:         "direct",
					Status:         "failed",
					Duration:       1800,
					QueuedDuration: 60,
					Variables: []Variables{
						{Key: "ENV", Value: "staging"},
					},
				},
				ParentPipeline: ParentPipeline{}, // No parent
				User: User{
					Name:     "Jane Doe",
					Username: "janedoe",
					Email:    "jane@example.com",
				},
				Commit: Commit{}, // No commit info
			},
			expected: map[string]string{
				conventionsAttributeCiCdPipelineUrl:            "https://gitlab.com/test-pipeline",
				conventionsAttributeCidCPipelineRunId:          "124",
				conventionsAttributeCiCdPipelineDuration:       "1800",
				conventionsAttributeCiCdPipelineQueuedDuration: "60",
				conventionsAttributeCiCdPipelineUser:           "Jane Doe",
				conventionsAttributeCiCdPipelineUsername:       "janedoe",
				conventionsAttributeCiCdPipelineUserEmail:      "jane@example.com",
				// Variable assertions
				fmt.Sprintf("%s.%s", conventionsAttributeCiCdPipelineVariable, "ENV"): "staging",
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
				Id:          789,
				Url:         "https://gitlab.com/test-job-success",
				Stage:       "test",
				Status:      "success",
				Environment: Environment{Name: "prod"},
				Runner: Runner{
					Id:          101,
					Description: "High performance runner",
					IsActive:    true,
					IsShared:    false,
					Tags:        []string{"docker", "linux"},
				},
			},
			expected: map[string]string{
				conventionsAttributeCiCdTaskRunId:            "789",
				conventionsAttributeCiCdTaskRunUrl:           "https://gitlab.com/test-job-success",
				conventionsAttributeCiCdPipelineTaskType:     "test",
				conventionsAttributeCiCdJobEnvironment:       "prod",
				conventionsAttributeCiCdJobRunnerId:          "101",
				conventionsAttributeCiCdJobRunnerDescription: "High performance runner",
				conventionsAttributeCiCdJobRunnerIsActive:    "true",
				conventionsAttributeCiCdJobRunnerIsShared:    "false",
				conventionsAttributeCiCdJobRunnerTag:         "docker",
			},
		},
		{
			name: "Failed job with runner tags",
			job: Job{
				Id:          790,
				Url:         "https://gitlab.com/test-job-fail",
				Stage:       "deploy",
				Status:      "failed",
				Environment: Environment{Name: "staging"},
				Runner: Runner{
					Id:          102,
					Description: "Backup runner",
					IsActive:    false,
					IsShared:    true,
					Tags:        []string{"backup", "windows"},
				},
			},
			expected: map[string]string{
				conventionsAttributeCiCdTaskRunId:            "790",
				conventionsAttributeCiCdTaskRunUrl:           "https://gitlab.com/test-job-fail",
				conventionsAttributeCiCdPipelineTaskType:     "deploy",
				conventionsAttributeCiCdJobEnvironment:       "staging",
				conventionsAttributeCiCdJobRunnerId:          "102",
				conventionsAttributeCiCdJobRunnerDescription: "Backup runner",
				conventionsAttributeCiCdJobRunnerIsActive:    "false",
				conventionsAttributeCiCdJobRunnerIsShared:    "true",
				conventionsAttributeCiCdJobRunnerTag:         "backup",
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

			// Special check for runner tags (since there may be multiple)
			for _, tag := range tt.job.Runner.Tags {
				if actualTagValue, exists := span.Attributes().Get(conventionsAttributeCiCdJobRunnerTag); !exists || actualTagValue.Str() != tag {
					t.Errorf("expected tag %s, but got %s", tag, actualTagValue.Str())
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

func TestDecode(t *testing.T) {
	req := &http.Request{
		Body: io.NopCloser(strings.NewReader(gitlabPipelineEvent)),
	}

	got, err := decode[glPipelineEvent](req)
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}

	want := glPipelineEvent{
		Kind: "pipeline",
		Pipeline: Pipeline{
			Id:     1234567890,
			Status: "pending",
		},
		Jobs: []Job{
			Job{
				Id:     7961245403,
				Name:   "job1",
				Status: "pending",
			},
		},
	}
	assert.Equal(t, want, got, "decoded result does not match expected")
}

func TestParseGitlabTime(t *testing.T) {
	nullTime, err := parseGitlabTime("null")
	if err != nil {
		t.Fatalf("expected no error for nullTime, but got: %v", err)
	}
	assert.Equal(t, nullTime, pcommon.Timestamp(0x0))

	emptyTime, err := parseGitlabTime("")
	if err != nil {
		t.Fatalf("expected no error for emptyTime, but got: %v", err)
	}
	assert.Equal(t, emptyTime, pcommon.Timestamp(0x0))

	validTime, err := parseGitlabTime(gitlabStartTime)
	if err != nil {
		t.Fatalf("expected no error for a valid time, but got: %v", err)
	}
	expectedTimestamp := pcommon.Timestamp(1704112215000000000)
	assert.Equal(t, validTime, expectedTimestamp)
}
