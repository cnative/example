package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cnative/pkg/health"
	"github.com/cnative/pkg/log"

	"github.com/cnative/pkg/auth"
	"github.com/cnative/pkg/server/middleware"

	"contrib.go.opencensus.io/exporter/ocagent"

	"contrib.go.opencensus.io/exporter/prometheus"
	grpc_runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.opencensus.io/zpages"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// default process metrics collection frequency
const defaultProcessMetricsCollectionFrequency = 5 * time.Second

type (

	// GRPCAPIHandler handles api registration with the grpc server
	GRPCAPIHandler interface {
		Register(context.Context, *grpc.Server, *grpc_runtime.ServeMux) error
		io.Closer
	}
	runtime struct {
		logger        *log.Logger
		probes        map[string]health.Probe
		grpcServer    *grpc.Server
		gwServer      *http.Server
		metricsServer *http.Server
		healthServer  health.Service
		debugServer   *http.Server
		htServer      *http.Server
		httpHandler   http.Handler

		authRuntime    auth.Runtime
		grpcAPIHandler GRPCAPIHandler

		gPort  uint // GRPC server port
		htPort uint // HTTP server port
		gwPort uint // gateway server port
		hPort  uint // health server port
		mPort  uint // metrics server port
		dPort  uint // debug server port

		certFile string
		keyFile  string
		clientCA string

		grpcEnabled      bool // enable grpc server
		htEnabled        bool // enable http server
		gwEnabled        bool // enable gateway server
		debugEnabled     bool // if enabled serve pprof data via HTTP server
		traceEnabled     bool
		ocAgentEP        string
		ocAgentNamespace string
		ocExporter       *ocagent.Exporter // ocexporter used only for tracing. will eventually use the same for stats as well

		pcm                   ProcessMetricsCollector
		processMetricsEnabled bool
		tags                  map[string]string // info purpose labels
		startTime             time.Time
		statsViews            []*view.View
	}

	//Runtime interface defines server operations
	Runtime interface {
		Start(context.Context) (chan error, error)
		Stop(context.Context)
	}
)

func (f optionFunc) apply(r *runtime) {
	f(r)
}

// NewRuntime returns a new Runtime
func NewRuntime(ctx context.Context, name string, options ...Option) (Runtime, error) {
	// setup defaults
	r := &runtime{}
	for _, opt := range options {
		opt.apply(r)
	}
	if r.logger == nil {
		r.logger, _ = log.NewNop()
	}

	var tlsConfig *tls.Config
	r.logger.Infow("TLS info", "key-file", r.keyFile, "cert-file", r.certFile, "client-ca", r.clientCA)
	if r.keyFile != "" && r.certFile != "" {
		var err error
		if tlsConfig, err = r.getTLSConfig(); err != nil {
			return nil, err
		}
	} else {
		r.logger.Errorf("no TLS key specified for servers. will start server insecurely....")
	}

	r.healthServer = health.New(health.BindPort(r.hPort), health.Logger(r.logger))
	metricsHandler := http.NewServeMux()
	r.metricsServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", r.mPort),
		Handler: metricsHandler,
	}
	r.registerPromMetricsExporter(metricsHandler)
	r.registerMetricsViews()

	if r.traceEnabled {
		// we use opencensus exporter only for trace. eventually we will use this for metrics as well
		r.logger.Infow("registering opencensus exporter", "agent-ep", r.ocAgentEP, "namespace", r.ocAgentNamespace)
		if err := r.registerOpencensusExporter(); err != nil {
			return nil, err
		}
	} else {
		r.logger.Warnf("tracing not enabled")
	}

	if r.debugEnabled {
		r.debugServer = &http.Server{
			Addr:    fmt.Sprintf("127.0.0.1:%d", r.dPort),
			Handler: getDebugHandler(r),
		}
	}

	if r.grpcEnabled {
		r.logger.Debug("creating grpc server")
		gsrv, err := r.newGRPCServerWithMetrics(tlsConfig)
		if err != nil {
			return nil, err
		}

		r.grpcServer = gsrv
		var gwmux *grpc_runtime.ServeMux
		if r.gwEnabled {
			r.logger.Info("grpc gateway enabled")
			gwmux = grpc_runtime.NewServeMux(grpc_runtime.WithMarshalerOption(grpc_runtime.MIMEWildcard, &grpc_runtime.JSONPb{EmitDefaults: true}))
			var h http.Handler
			h = gwmux
			if r.authRuntime != nil {
				// auth runtime set
				h = middleware.HTTPBearerTokenAuth(r.authRuntime, gwmux)
			} else {
				r.logger.Error("auth runtime not enabled for the grpc gateway server")
			}
			r.gwServer = &http.Server{
				Addr:      fmt.Sprintf(":%d", r.gwPort),
				Handler:   &ochttp.Handler{Handler: h}, // instruments http with opencensus
				TLSConfig: tlsConfig,
			}
		} else {
			r.logger.Info("grpc gateway not enabled")
		}

		if err := r.grpcAPIHandler.Register(ctx, r.grpcServer, gwmux); err != nil {
			return nil, err
		}
	}

	if r.htEnabled {
		r.logger.Info("http server enabled")
		r.htServer = &http.Server{
			Addr:      fmt.Sprintf(":%d", r.htPort),
			Handler:   &ochttp.Handler{Handler: r.httpHandler},
			TLSConfig: tlsConfig,
		}
	}

	return r, nil
}

// Start server runtime
func (r *runtime) Start(ctx context.Context) (chan error, error) {

	errc := make(chan error, 7) // error buffer channel as we have about 7 goroutines below

	// Shutdown on SIGINT, SIGTERM
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// Start process metrics collector
	if r.processMetricsEnabled {
		r.pcm = NewProcessMetricsCollector()
		go func() {
			r.logger.Info("starting process metrics collector")
			_ = r.pcm.Start()
		}()
	} else {
		r.logger.Warn("skipping process metrics collection")
	}

	// Start http listener that exposes server pprof runtime data
	if r.debugEnabled {
		go func() {
			r.logger.Infow("starting debug server", "port", r.dPort)
			err := r.debugServer.ListenAndServe()
			errc <- errors.Wrap(err, "debug server returned an error")
		}()
	}

	if r.grpcEnabled {
		// start gRPC server
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", r.gPort))
		if err != nil {
			r.logger.Errorf("failed to create grpc listener -%v ", err)
			return nil, err
		}
		go func() {
			r.logger.Infow("starting grpc server", "port", r.gPort)
			err := r.grpcServer.Serve(lis)
			errc <- errors.Wrap(err, "server returned an error")
		}()
	}

	if r.gwEnabled {
		// start gRPC gateway
		go func() {
			r.logger.Infow("starting gateway server", "port", r.gwPort)
			err := r.gwServer.ListenAndServeTLS(r.certFile, r.keyFile)
			errc <- errors.Wrap(err, "gateway server returned an error")
		}()
	}

	if r.htEnabled {
		// start HTTP server
		go func() {
			r.logger.Infow("starting http server", "port", r.htPort)
			var err error
			if r.certFile == "" && r.keyFile == "" {
				err = r.htServer.ListenAndServe()
			} else {
				err = r.htServer.ListenAndServeTLS(r.certFile, r.keyFile)
			}
			errc <- errors.Wrap(err, "http server returned an error")
		}()
	}

	// Start health server
	go func() {
		r.logger.Infow("starting health service", "port", r.hPort)
		for name, probe := range r.probes {
			r.healthServer.RegisterProbe(name, probe)
		}
		err := r.healthServer.Start()
		errc <- errors.Wrap(err, "health service returned an error")
	}()

	// Start metrics server
	go func() {
		r.logger.Infow("starting metrics server", "port", r.mPort)
		err := r.metricsServer.ListenAndServe()
		errc <- errors.Wrap(err, "metrics service returned an error")
	}()

	r.startTime = time.Now()
	return errc, nil
}

// Stop server runtime
func (r *runtime) Stop(ctx context.Context) {

	r.logger.Infof("shutting down..")
	if r.grpcAPIHandler != nil {
		r.grpcAPIHandler.Close()
	}

	if r.grpcEnabled {
		// gracefully shutdown the gRPC server
		r.logger.Info("shutting grpc server")
		r.grpcServer.GracefulStop()
	}

	if r.gwEnabled {
		r.logger.Info("shutting gateway server")
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := r.gwServer.Shutdown(ctx); err != nil {
			r.logger.Errorf("error happened while shutting gateway server -%v", err)
		}
	}

	if r.htEnabled {
		r.logger.Info("shutting HTTP server")
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := r.htServer.Shutdown(ctx); err != nil {
			r.logger.Errorf("error happened while shutting HTTP server -%v", err)
		}
	}

	// gracefully shutdown the health server
	r.logger.Info("shutting health server")
	if err := r.healthServer.Stop(ctx); err != nil {
		r.logger.Fatalf("error shutting down health server %v ", err)
	}

	if r.debugEnabled {
		r.logger.Info("shutting debug server")
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := r.debugServer.Shutdown(ctx); err != nil {
			r.logger.Errorf("error happened while shutting debug server -%v", err)
		}
	}

	r.logger.Info("shutting metrics server")
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := r.metricsServer.Shutdown(ctx); err != nil {
		r.logger.Errorf("error happened while shutting metrics server -%v", err)
	}

	if r.processMetricsEnabled {
		// stop collecting process metrics
		r.pcm.Stop()
	}

	if r.traceEnabled {
		r.logger.Info("stopping opencensus exporter")
		if err := r.ocExporter.Stop(); err != nil {
			r.logger.Errorf("error happened while stopping oc exporter", err)
		}
	}
}

func (r *runtime) newGRPCServerWithMetrics(tlsConfig *tls.Config) (*grpc.Server, error) {
	r.logger.Debug("creating new gRPC server with default server metrics views")

	opts := []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
			MaxConnectionAge:      30 * time.Second, // If any connection is alive for more than 30 seconds, send a GOAWAY
			MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
			Time:                  5 * time.Second,  // Ping the client if it is idle for 5 seconds to ensure the connection is still active
			Timeout:               1 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
			PermitWithoutStream: true,            // Allow pings even when there are no active streams
		}),
	}
	if r.authRuntime != nil {
		opts = append(opts, middleware.GRPCAuth(r.authRuntime)...)
	} else {
		r.logger.Error("auth runtime not enabled for the grpc server")
	}

	if tlsConfig != nil {
		// Create the TLS credentials
		cred := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.Creds(cred))
	}

	return grpc.NewServer(opts...), nil
}

// register trace exporter
func (r *runtime) registerOpencensusExporter() (err error) {

	r.ocExporter, err = ocagent.NewExporter(
		ocagent.WithInsecure(),
		ocagent.WithReconnectionPeriod(5*time.Second),
		ocagent.WithAddress(r.ocAgentEP),
		ocagent.WithServiceName(r.ocAgentNamespace))

	if err != nil {
		r.logger.Fatalf("failed to create ocagent-exporter: %v", err)
		return err
	}

	trace.RegisterExporter(r.ocExporter)
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample(),
		MaxAttributesPerSpan:       trace.DefaultMaxAttributesPerSpan,
		MaxAnnotationEventsPerSpan: trace.DefaultMaxAnnotationEventsPerSpan,
		MaxMessageEventsPerSpan:    trace.DefaultMaxMessageEventsPerSpan,
		MaxLinksPerSpan:            trace.DefaultMaxLinksPerSpan})

	return nil
}

// registers prometheus metrics exporter
func (r *runtime) registerPromMetricsExporter(mux *http.ServeMux) {

	// create the Prometheus exporter.
	pe, err := prometheus.NewExporter(prometheus.Options{})
	if err != nil {
		r.logger.Fatalf("Failed to create prometheus metrics exporter: %v", err)
	}

	view.RegisterExporter(pe)
	r.logger.Debug("registering prometheus exporter with http server mux")

	mux.Handle("/metrics", pe)
	zpages.Handle(mux, "/")
}

// get TLS Config
func (r *runtime) getTLSConfig() (*tls.Config, error) {
	// Load the certificates from disk
	certificate, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := tls.Config{
		Certificates: []tls.Certificate{certificate},
	}

	if r.clientCA != "" {
		// Create a certificate pool from the certificate authority
		certPool := x509.NewCertPool()
		ca, err := ioutil.ReadFile(r.clientCA)
		if err != nil {
			return nil, err
		}

		// Append the client certificates from the CA
		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			return nil, err
		}
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.ClientCAs = certPool
	} else {
		r.logger.Errorf("mTLS not enabled")
	}

	return &tlsConfig, nil
}

func (r *runtime) registerMetricsViews() {

	// process stats
	if err := view.Register(DefaultProcessViews...); err != nil {
		r.logger.Fatalf("failed to register default process views: %v", err)
	}

	// grpc server stats
	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		r.logger.Fatalf("failed to register ocgrpc server views: %v", err)
	}

	// http server stats
	if err := view.Register(ochttp.DefaultServerViews...); err != nil {
		r.logger.Fatalf("failed to register ocgrpc server views: %v", err)
	}

	// custom stats
	if err := view.Register(r.statsViews...); err != nil {
		r.logger.Fatalf("Failed to register ocgrpc server views: %v", err)
	}
}
