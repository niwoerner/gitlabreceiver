package gitlabreceiver

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type gitlabReceiver struct {
	host               component.Host
	cancel             context.CancelFunc
	cfg                *Config
	logger             *zap.Logger
	nextTracesConsumer consumer.Traces
	httpServer         *http.Server
	settings           *receiver.Settings
	shutdownWG         sync.WaitGroup
	glResource         gitlabResource
}

func newGitlabReceiver(cfg component.Config, s receiver.Settings) *gitlabReceiver {
	return &gitlabReceiver{
		logger:   s.Logger,
		settings: &s,
		cfg:      cfg.(*Config),
	}
}

func (glRcvr *gitlabReceiver) Start(ctx context.Context, host component.Host) error {
	glRcvr.host = host
	ctx, glRcvr.cancel = context.WithCancel(ctx)

	interval, _ := time.ParseDuration(glRcvr.cfg.Interval)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		err := glRcvr.startHTTPServer(ctx, host)
		if err != nil {
			glRcvr.logger.Panic("Unable to start", zap.Error(err))
		}
		//ToDo: Remove interval loop - only used for logging output for now
		for {
			select {
			case <-ticker.C:
				glRcvr.logger.Info("The gitlab receiver is running")
			}
		}
	}()
	return nil
}

func (glRcvr *gitlabReceiver) Shutdown(ctx context.Context) error {
	if glRcvr.httpServer != nil {
		err := glRcvr.httpServer.Shutdown(ctx)
		return err
	}
	glRcvr.shutdownWG.Wait()
	return nil
}

func (glRcvr *gitlabReceiver) startHTTPServer(ctx context.Context, host component.Host) error {
	var err error
	httpMux := http.NewServeMux()
	glRcvr.httpServer, err = glRcvr.cfg.ToServer(ctx, host, glRcvr.settings.TelemetrySettings, httpMux)
	if err != nil {
		return err
	}

	listener, err := glRcvr.cfg.ServerConfig.ToListener(ctx)
	if err != nil {
		return err
	}

	if glRcvr.nextTracesConsumer != nil {
		httpMux.HandleFunc(glRcvr.cfg.Traces.UrlPath, func(resp http.ResponseWriter, req *http.Request) {
			glRcvr.handleTraces(ctx, resp, req)
		})
	}

	glRcvr.logger.Info("Starting HTTP Server", zap.String("endpoint", glRcvr.cfg.Endpoint))

	glRcvr.shutdownWG.Add(1)
	go func() {
		defer glRcvr.shutdownWG.Done()
		if err := glRcvr.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			glRcvr.logger.Error("Error starting HTTP server", zap.String("error", err.Error()))
		}
	}()

	return nil
}

func (glRcvr *gitlabReceiver) handleTraces(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	err := glRcvr.validateReq(req)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		glRcvr.logger.Error("Invalid request - Validation failed", zap.Error(err))
		return
	}

	glEvent, err := glRcvr.unmarshalReq(req)
	if err != nil {
		http.Error(w, "Unable to handle the request", http.StatusBadRequest)
		glRcvr.logger.Error("Error unmarshalling the request", zap.Error(err))
		return
	}

	glPipelineEvent := glEvent.(*glPipelineEvent)
	glRcvr.glResource = glPipelineEvent

	if len(glRcvr.cfg.Traces.Refs) > 0 && !slices.Contains(glRcvr.cfg.Traces.Refs, glPipelineEvent.Pipeline.Ref) {
		glRcvr.logger.Info("Received ref is not configured to be exported.", zap.String("Pipeline", glPipelineEvent.Pipeline.Url), zap.String("Ref", glPipelineEvent.Pipeline.Ref))
		_, err = w.Write([]byte("Not configured to be exported"))
		if err != nil {
			glRcvr.logger.Error("Unable to send response", zap.Error(err))
		}
		return
	}

	// we only want to export the root span if the pipeline is finished
	//finished date and running status would inidcate some sort of retry/restart which we want to export once it is finished in a separate trace
	if glPipelineEvent.Pipeline.FinishedAt != "" && glPipelineEvent.Pipeline.Status != "running" {
		err = glRcvr.exportTraces(ctx)
		if err != nil {
			http.Error(w, "Unable to export the trace", http.StatusInternalServerError)
			glRcvr.logger.Error("Unable to export the trace", zap.Error(err))
			return
		}
	}

	_, err = w.Write([]byte("OK"))
	if err != nil {
		glRcvr.logger.Error("Unable to send response", zap.Error(err))
	}
}

func (glRcvr *gitlabReceiver) validateReq(req *http.Request) error {
	if req.Method != http.MethodPost {
		return errors.New("invalid HTTP method")
	}

	if req.Header.Get("Content-Type") != "application/json" {
		return errors.New("request has unsupported content type")
	}

	if req.Header.Get("X-Gitlab-Event") != "Pipeline Hook" {
		return errors.New("invalid request header")
	}

	return nil
}

// ToDo: Refactor the unmarshal process when there is more than one possible event
func (glRcvr *gitlabReceiver) unmarshalReq(req *http.Request) (gitlabResource, error) {
	var glEvent gitlabResource
	var err error
	if req.Header.Get("X-Gitlab-Event") == "Pipeline Hook" {
		glEvent, err = decode[*glPipelineEvent](req)
		if err != nil {
			return nil, errors.New("unable to read the body: " + err.Error())
		}
	}

	return glEvent, nil
}

func (glRcvr *gitlabReceiver) exportTraces(ctx context.Context) error {
	traces, err := glRcvr.glResource.newTrace()
	if err != nil {
		return err
	}

	err = glRcvr.nextTracesConsumer.ConsumeTraces(ctx, *traces)
	if err != nil {
		return err
	}

	return nil
}
