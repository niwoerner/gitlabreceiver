package gitlabreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
}

func TestSanitizeUrlPath(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedResult string
	}{
		{
			name:           "to be sanitzed url",
			url:            "/xyz?someParams",
			expectedResult: "/xyz",
		},
		{
			name:           "absolute url",
			url:            "xyz",
			expectedResult: "/xyz",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := &Config{
				TracesURLPath: tc.url,
			}
			u, err := sanitizeURLPath(config.TracesURLPath)
			require.NoError(t, err, "error sanitizing url")
			assert.Equal(t, tc.expectedResult, u, "Must match the expected url")
		})
	}
}
