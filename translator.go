package gitlabreceiver

import (
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/rand"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// We use the first 16 bytes from the generated hash
// Details: https://www.w3.org/TR/trace-context/#traceparent-header-field-values
func getTraceId(commitSHA string, pipelineId string, endTime string) ([16]byte, error) {
	var traceId [16]byte

	hash, err := generateHash(commitSHA, pipelineId, endTime)
	if err != nil {
		return traceId, err
	}
	copy(traceId[:], hash[:16])

	return traceId, nil
}

// We use 16-24bytes from the generated hash (0-16 used for the TraceId)
// Details: https://www.w3.org/TR/trace-context/#traceparent-header-field-values
func getRootSpanId(commitSHA string, pipelineId string, endTime string) ([8]byte, error) {
	var spanId [8]byte

	hash, err := generateHash(commitSHA, pipelineId, endTime)
	if err != nil {
		return spanId, err
	}
	copy(spanId[:], hash[16:24])

	return spanId, nil
}

// We generate the hash based on commitSHA, pipelineId and endTime of the pipeline. This gives us an unique hash for every pipeline which needs to be exported.
// It is important to consider the endTime, because otherwise a retried, finished pipeline would have the same TraceId/SpanId if we don't take the finsished time into account
func generateHash(commitSHA string, pipelineId string, endTime string) ([32]byte, error) {
	if commitSHA == "" || pipelineId == "" || endTime == "" {
		return [32]byte{}, errors.New("commitSHA, pipelineId, and endTime must be non-empty")
	}
	return sha256.Sum256([]byte(commitSHA + pipelineId + endTime)), nil
}

func getRandomSpanId() pcommon.SpanID {
	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)
	randSource := rand.New(rand.NewSource(rngSeed))

	var sid [8]byte
	randSource.Read(sid[:])
	spanID := pcommon.SpanID(sid)

	return spanID
}

func createSpan(rs ptrace.ResourceSpans, traceId [16]byte, spanId [8]byte, parentSpanId [8]byte, name string, startTime pcommon.Timestamp, endTime pcommon.Timestamp, glRes gitlabResource) {
	scopeSpanSlice := rs.ScopeSpans()
	scopeSpanSlice.EnsureCapacity(1)
	ss := scopeSpanSlice.AppendEmpty()
	span := ss.Spans().AppendEmpty()

	span.SetTraceID(pcommon.TraceID(traceId))
	span.SetSpanID(spanId)
	if parentSpanId != [8]byte{0, 0, 0, 0, 0, 0, 0, 0} {
		span.SetParentSpanID(parentSpanId)
	}

	span.SetStartTimestamp(startTime)
	span.SetEndTimestamp(endTime)

	span.SetName(name)
	glRes.setAttributes(span)
}
