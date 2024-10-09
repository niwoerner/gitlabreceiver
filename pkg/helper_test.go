package gitlabreceiver

import (
	"crypto/sha256"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	//usual state after pipeline creation
	pipelineCreatedJobPending = `{
		"object_kind": "pipeline",
		"object_attributes": {
			"id": 1234567890
		},
		"builds": [
			{
				"id": 7961245403,
				"stage": "stage1",
				"name": "job1",
				"status": "pending"
			},			{
				"id": 7961245403,
				"stage": "stage2",
				"name": "job2",
				"status": "pending"
			},			{
				"id": 7961245403,
				"stage": "stage3",
				"name": "job3",
				"status": "pending"
			}
		]
	}`
)

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

func TestNewSpanId(t *testing.T) {
	spanId1 := newSpanId()
	assert.Len(t, spanId1, 8, "span ID should be 8 bytes long")

	spanId2 := newSpanId()
	assert.NotEqual(t, spanId1, spanId2, "span IDs should be unique")
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

func TestGenerateTraceParentHeader(t *testing.T) {
	tests := []struct {
		commitSHA        string
		pipelineId       string
		expectedTraceId  [16]byte
		expectedParentId [8]byte
		duration         string
	}{
		{
			commitSHA:        "abc123",
			pipelineId:       "pipeline1",
			expectedTraceId:  generateExpectedTraceId("abc123", "pipeline1", "10"),
			expectedParentId: generateExpectedSpanId("abc123", "pipeline1", "10"),
			duration:         "10",
		},
		{
			commitSHA:        "def456",
			pipelineId:       "pipeline2",
			expectedTraceId:  generateExpectedTraceId("def456", "pipeline2", "10"),
			expectedParentId: generateExpectedSpanId("def456", "pipeline2", "10"),
			duration:         "10",
		},
	}
	for _, tt := range tests {
		traceId, parentId := generateTraceParentHeader(tt.commitSHA, tt.pipelineId, tt.duration)

		if traceId != tt.expectedTraceId {
			t.Errorf("generateTraceParentHeader(%q, %q) = traceId %v; want %v", tt.commitSHA, tt.pipelineId, traceId, tt.expectedTraceId)
		}

		if parentId != tt.expectedParentId {
			t.Errorf("generateTraceParentHeader(%q, %q) = parentId %v; want %v", tt.commitSHA, tt.pipelineId, parentId, tt.expectedParentId)
		}
	}

	traceId1, parentId1 := generateTraceParentHeader("testSHA", "testPipeline", "10")
	traceId2, parentId2 := generateTraceParentHeader("testSHA", "testPipeline", "10")

	if traceId1 != traceId2 {
		t.Errorf("Expected trace IDs to be equal, but got %v and %v", traceId1, traceId2)
	}
	if parentId1 != parentId2 {
		t.Errorf("Expected parent IDs to be equal, but got %v and %v", parentId1, parentId2)
	}
}

func generateExpectedTraceId(commitSHA string, pipelineId string, duration string) [16]byte {
	hash := sha256.Sum256([]byte(commitSHA + pipelineId + duration))
	var traceId [16]byte
	copy(traceId[:], hash[:16])
	return traceId
}

func generateExpectedSpanId(commitSHA string, pipelineId string, duration string) [8]byte {
	hash := sha256.Sum256([]byte(commitSHA + pipelineId + duration))
	var parentId [8]byte
	copy(parentId[:], hash[16:24])
	return parentId
}
