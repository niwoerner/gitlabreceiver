package gitlabreceiver

import (
	"fmt"
	"net/url"
	"path"
	"time"

	"go.opentelemetry.io/collector/confmap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
)

const (
	defaultInterval      = 10 * time.Second
	defaultTracesUrlPath = "/v0.1/traces"
)

var typeStr = component.MustNewType("gitlab")

type GitlabPath struct {
	Refs []string `mapstructure:"refs"`
}
type Traces struct {
	UrlPath string     `mapstructure:"url_path,omitempty"`
	Path    GitlabPath `mapstructure:"path"`
}
type Config struct {
	Interval                string `mapstructure:"interval"`
	confighttp.ServerConfig `mapstructure:",squash"`
	TracesURLPath           string            `mapstructure:"traces_url_path,omitempty"`
	Traces                  map[string]Traces `mapstructure:"traces"`
}

// ToDo: Add validation once needed
func (cfg *Config) Validate() error {
	return nil
}

func createDefaultConfig() component.Config {
	return &Config{
		Interval: defaultInterval.String(),
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "localhost:9286",
		},
		TracesURLPath: defaultTracesUrlPath,
	}
}

func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	err := conf.Unmarshal(cfg)
	if err != nil {
		return err
	}
	cfg.TracesURLPath, err = sanitizeURLPath(cfg.TracesURLPath)
	if err != nil {
		return err
	}
	return nil
}

func sanitizeURLPath(urlPath string) (string, error) {
	u, err := url.Parse(urlPath)
	if err != nil {
		return "", fmt.Errorf("invalid HTTP URL path: %w", err)
	}

	if !path.IsAbs(u.Path) {
		u.Path = "/" + u.Path
	}
	return u.Path, nil
}
