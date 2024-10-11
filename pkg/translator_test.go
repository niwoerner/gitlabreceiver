package gitlabreceiver

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestgetRandomSpanId(t *testing.T) {
	spanId1 := getRandomSpanId()
	assert.Len(t, spanId1, 8, "span ID should be 8 bytes long")

	spanId2 := getRandomSpanId()
	assert.NotEqual(t, spanId1, spanId2, "span IDs should be unique")
}

func TestGetTraceAndSpanIds(t *testing.T) {
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
		traceId, err := getTraceId(tt.commitSHA, tt.pipelineId, tt.duration)
		parentId, err := getRootSpanId(tt.commitSHA, tt.pipelineId, tt.duration)
		assert.NoError(t, err)

		if traceId != tt.expectedTraceId {
			t.Errorf("generateTraceParentHeader(%q, %q) = traceId %v; want %v", tt.commitSHA, tt.pipelineId, traceId, tt.expectedTraceId)
		}

		if parentId != tt.expectedParentId {
			t.Errorf("generateTraceParentHeader(%q, %q) = parentId %v; want %v", tt.commitSHA, tt.pipelineId, parentId, tt.expectedParentId)
		}
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
