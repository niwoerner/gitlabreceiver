package gitlabreceiver

import (
	"errors"
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
	gitlabPathPrefix     = "path-"
)

var typeStr = component.MustNewType("gitlab")

type Traces struct {
	UrlPath string   `mapstructure:"url_path,omitempty"`
	Refs    []string `mapstructure:"refs,omitempty"`
}

type Config struct {
	Interval                string `mapstructure:"interval"`
	confighttp.ServerConfig `mapstructure:",squash"`
	Traces                  Traces `mapstructure:"traces"`
}

func (cfg *Config) Validate() error {
	if len(cfg.Traces.Refs) > 50 {
		return errors.New("configured amount of refs is exceeding the limit of 50")
	}
	return nil
}

func createDefaultConfig() component.Config {
	return &Config{
		Interval: defaultInterval.String(),
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "localhost:9286",
		},
		Traces: Traces{
			UrlPath: defaultTracesUrlPath,
			Refs:    []string{},
		},
	}
}

func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	err := conf.Unmarshal(cfg)
	if err != nil {
		return err
	}
	cfg.Traces.UrlPath, err = sanitizeURLPath(cfg.Traces.UrlPath)
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
