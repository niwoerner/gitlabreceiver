package gitlabreceiver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestGitlabReceiverHttpServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel) //clean up allocated resource when the test is finished

	s := receivertest.NewNopSettings()
	host := componenttest.NewNopHost()
	cfg, ok := createDefaultConfig().(*Config)
	if !ok {
		require.True(t, ok, "invalid input type")
	}
	p, err := getFreePort()
	require.NoError(t, err, "error finding an available port")
	cfg.Protocols.HTTP.Endpoint = fmt.Sprintf("localhost:%s", p)

	glRcvr := newGitlabReceiver(cfg, s)
	glRcvr.nextTracesConsumer = consumertest.NewNop()

	require.NoError(t, err, "Failed to create traces receiver")
	require.NoError(t, glRcvr.Start(ctx, host), "failed to start http server")
	t.Cleanup(func() {
		require.NoError(t, glRcvr.Shutdown(ctx), "failed to shutdown http server")
	})

	tests := []struct {
		name       string
		httpMethod string
		reqBody    []byte
		resBody    string
		statusCode int
	}{
		{
			name:       "unsupported httpMethod",
			httpMethod: http.MethodGet,
			reqBody:    nil,
			resBody:    "Invalid request\n",
			statusCode: http.StatusBadRequest,
		}, {
			name:       "invalid requestBody",
			httpMethod: http.MethodPost,
			reqBody:    []byte("invalid req body"),
			resBody:    "Unable to handle the request\n",
			statusCode: http.StatusBadRequest,
		}, {
			name:       "valid request",
			httpMethod: http.MethodPost,
			reqBody:    []byte(pipelineCreatedJobPending),
			resBody:    "OK",
			statusCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequest(tc.httpMethod, fmt.Sprintf("http://%s%s", cfg.HTTP.Endpoint, cfg.HTTP.TracesURLPath), bytes.NewReader(tc.reqBody))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Gitlab-Event", "Pipeline Hook")
			require.NoError(t, err, "Unable to create a request")

			resp, err := http.DefaultClient.Do(request)
			require.NoError(t, err, "Error sending request")

			resBody, err := io.ReadAll(resp.Body)
			require.NoError(t, errors.Join(err, resp.Body.Close()), "Error reading response body")
			assert.Equal(t, tc.resBody, string(resBody))
			assert.Equal(t, tc.statusCode, resp.StatusCode, "Must match the expected status code")
		})
	}
}

func getFreePort() (string, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", fmt.Errorf("failed to listen on a random port: %w", err)
	}
	defer listener.Close()
	return strconv.Itoa(listener.Addr().(*net.TCPAddr).Port), nil
}
