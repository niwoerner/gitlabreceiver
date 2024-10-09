package gitlabreceiver

import (
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func decode[T any](r *http.Request) (T, error) {
	var v T
	err := json.NewDecoder(r.Body).Decode(&v)
	if err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

func newSpanId() pcommon.SpanID {
	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)
	randSource := rand.New(rand.NewSource(rngSeed))

	var sid [8]byte
	randSource.Read(sid[:])
	spanID := pcommon.SpanID(sid)

	return spanID
}

func parseGitlabTime(t string) (pcommon.Timestamp, error) {
	if t == "" || t == "null" {
		return 0, nil
	}

	//For some reason the gitlab test pipeline event has a different time format which we need to support to test (and eventually reenable webhooks) therefoe we are continuing on error to handle the webhook test and the actual webhook
	pt, err := time.Parse(gitlabEventTimeFormat, t)
	if err == nil {
		return pcommon.NewTimestampFromTime(pt), nil
	}

	pt, err = time.Parse(time.RFC3339, t) //Time format of test pipeline events
	if err == nil {
		return pcommon.NewTimestampFromTime(pt), nil
	}

	//This return reflects the error case, not the expected case like usually
	return 0, err

}

// To propagate context, we need to have a traceparent header set in our requests as per the w3c spec: https://www.w3.org/TR/trace-context/#examples-of-http-traceparent-headers
// Details: https://www.w3.org/TR/trace-context/#traceparent-header-field-values
func generateTraceParentHeader(commitSHA string, pipelineId string, duration string) ([16]byte, [8]byte) {
	// var version byte = 0b00000000
	var traceId [16]byte
	var parentId [8]byte
	// var traceFlag byte = 0b00000001

	hash := sha256.Sum256([]byte(commitSHA + pipelineId + duration))
	copy(traceId[:], hash[:16])
	copy(parentId[:], hash[16:24])

	// traceIdHex = hex.EncodeToString(traceId[:])
	// parentIdHex = hex.EncodeToString(parentId[:])
	// versionHex = fmt.Sprintf("%02x", version)
	// traceFlagHex = fmt.Sprintf("%02x", traceFlag)
	// traceParent = fmt.Sprintf("%s-%s-%s-%s", versionHex, traceIdHex, parentIdHex, traceFlagHex)
	return traceId, parentId
}

func createSpan(rs ptrace.ResourceSpans, traceId [16]byte, spanId [8]byte, parentSpanId [8]byte, name string, status string, startTime pcommon.Timestamp, endTime pcommon.Timestamp, glRes gitlabResource) {
	resourceScopeSpans := rs.ScopeSpans().AppendEmpty() //resourceScopeSpans = hierachy between different spans within a trace
	span := resourceScopeSpans.Spans().AppendEmpty()
	span.SetTraceID(pcommon.TraceID(traceId))
	span.SetSpanID(spanId)
	if parentSpanId != [8]byte{0, 0, 0, 0, 0, 0, 0, 0} {
		span.SetParentSpanID(parentSpanId)
	}
	span.SetName(name)
	span.Status().SetMessage(status)

	span.SetStartTimestamp(startTime)
	span.SetEndTimestamp(endTime)

	glRes.setAttributes(span)
}
