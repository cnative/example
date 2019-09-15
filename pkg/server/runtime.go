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

	grpc_runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/cnative/example/pkg/auth"
	"github.com/cnative/example/pkg/health"
	"github.com/cnative/example/pkg/log"
	"github.com/cnative/example/pkg/server/middleware"
	"github.com/cnative/example/pkg/state"

	"contrib.go.opencensus.io/exporter/jaeger"
	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/pkg/errors"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.opencensus.io/zpages"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// default process metrics collection frequency
const defaultProcessMetricsCollectionFrequency = 5 * time.Second
const defaultPProfilePort = 6060

type (

	// GrpcHandler handles api registration with the grpc server
	GrpcHandler interface {
		Register(context.Context, *grpc.Server, *grpc_runtime.ServeMux) error
		io.Closer
	}

	runtime struct {
		logger        *log.Logger
		store         state.Store
		grpcServer    *grpc.Server
		gwServer      *http.Server
		metricsServer *http.Server
		healthServer  health.Service

		authRuntime auth.Runtime
		apiHandler  GrpcHandler

		port   uint // GRPC server port
		hPort  uint // Health server port
		mPort  uint // metrics server port
		gwPort uint // gateway server port

		certFile string
		keyFile  string
		clientCA string

		gwEnabled      bool
		traceEnabled   bool
		traceEndpoint  string
		traceNamespace string
		traceBackend   string

		pcm                   ProcessMetricsCollector
		processMetricsEnabled bool
		pprofEnabled          bool
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
func NewRuntime(otions ...Option) (Runtime, error) {
	r := &runtime{}
	for _, opt := range otions {
		opt.apply(r)
	}
	if r.logger == nil {
		r.logger, _ = log.NewNop()
	}

	metricsHandler := http.NewServeMux()
	r.metricsServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", r.mPort),
		Handler: metricsHandler,
	}

	r.registerMetricsViews()
	r.registerPromMetricsExporter(metricsHandler)
	r.registerTraceExporter()

	r.healthServer = health.New(health.BindPort(r.hPort), health.Logger(r.logger))
	r.healthServer.RegisterProbe("store", r.store)

	var tlsConfig *tls.Config
	var err error
	r.logger.Infow("TLS info", "key-file", r.keyFile, "cert-file", r.certFile, "client-ca", r.clientCA)
	if r.keyFile != "" && r.certFile != "" {
		if tlsConfig, err = r.getTLSConfig(); err != nil {
			return nil, err
		}
	} else {
		r.logger.Errorf("no TLS key specified for servers. will start server insecurely....")
	}

	r.logger.Debug("creating grpc server")
	gsrv, err := r.newGRPCServerWithMetrics(tlsConfig)
	if err != nil {
		return nil, err
	}
	r.grpcServer = gsrv
	ctx := context.Background()
	var gwmux *grpc_runtime.ServeMux
	if r.gwEnabled {
		r.logger.Info("grpc gateway enabled")
		gwmux = grpc_runtime.NewServeMux(grpc_runtime.WithMarshalerOption(grpc_runtime.MIMEWildcard, &grpc_runtime.JSONPb{EmitDefaults: true}))
		r.gwServer = &http.Server{
			Addr:      fmt.Sprintf(":%d", r.gwPort),
			Handler:   &ochttp.Handler{Handler: gwmux},
			TLSConfig: tlsConfig,
		}
	} else {
		r.logger.Info("grpc gateway not enabled")
	}

	if r.apiHandler != nil {
		if err := r.apiHandler.Register(ctx, r.grpcServer, gwmux); err != nil {
			return nil, err
		}
	}

	if r.processMetricsEnabled {
		r.pcm = NewProcessMetricsCollector()
		// Start process metrics collector
		go func() {
			r.logger.Info("starting process metrics collector")
			_ = r.pcm.Start() //TODO clean this up as part of introducing oklog/run group for start method below
		}()
	} else {
		r.logger.Warn("skipping process metrics collection")
	}

	return r, nil
}

// Start server runtime
func (r *runtime) Start(ctx context.Context) (chan error, error) {

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", r.port))
	if err != nil {
		r.logger.Errorf("Failed to listen -%v ", err)
		return nil, err
	}

	errc := make(chan error, 4)

	// Shutdown on SIGINT, SIGTERM
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// Start gRPC server
	go func() {
		r.logger.Infow("starting server", "port", r.port)
		err := r.grpcServer.Serve(lis)
		errc <- errors.Wrap(err, "server returned an error")
	}()

	if r.gwEnabled {
		// Start Gateway server
		go func() {
			r.logger.Infow("starting gateway server", "gw-port", r.gwPort)
			err := r.gwServer.ListenAndServe()
			errc <- errors.Wrap(err, "gateway returned an error")
		}()
	}

	// Start health server
	go func() {
		r.logger.Infow("starting health server", "port", r.hPort)
		err := r.healthServer.Start()
		errc <- errors.Wrap(err, "health server returned an error")
	}()

	// Start metrics server
	go func() {
		r.logger.Infow("starting metrics server", "port", r.mPort)
		err := r.metricsServer.ListenAndServe()
		errc <- errors.Wrap(err, "metrics server returned an error")
	}()

	// expose profile data via HTTP end point
	go func() {
		r.logger.Infow("expose HTTP server runtime profiling data ", "port", defaultPProfilePort)
		err := http.ListenAndServe(fmt.Sprintf(":%d", defaultPProfilePort), nil)
		errc <- errors.Wrap(err, "gateway returned an error")
	}()

	return errc, nil
}

// Stop server runtime
func (r *runtime) Stop(ctx context.Context) {

	r.logger.Infof("shutting down..")
	if err := r.apiHandler.Close(); err != nil {
		r.logger.Fatalf("error shutting down grpc server handler %v ", err)
	}

	// gracefully shutdown the health server
	r.logger.Info("shutting healh server")
	if err := r.healthServer.Stop(context.Background()); err != nil {
		r.logger.Fatalf("error shutting down health server %v ", err)
	}

	if r.gwEnabled {
		// gracefully shutdown the Gateway server
		r.logger.Info("shutting gateway server")
		ctx, cancel1 := context.WithTimeout(ctx, 30*time.Second)
		defer cancel1()
		err := r.gwServer.Shutdown(ctx)
		if err != nil {
			r.logger.Errorf("An error happened while shutting down -%v", err)
		}
	}

	r.logger.Info("shutting metrics server")
	ctx, cancel2 := context.WithTimeout(ctx, 30*time.Second)
	defer cancel2()
	if err := r.metricsServer.Shutdown(ctx); err != nil {
		r.logger.Errorf("error happened while shutting down -%v", err)
	}

	if r.processMetricsEnabled {
		// stop collecting process metrics
		r.pcm.Stop()
	}

	// Gracefully shutdown the gRPC server
	r.grpcServer.GracefulStop()
}

func (r *runtime) newGRPCServerWithMetrics(tlsConfig *tls.Config) (*grpc.Server, error) {

	opts := []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
	}
	if r.authRuntime != nil {
		opts = append(opts, middleware.Auth(r.authRuntime)...)
	} else {
		r.logger.Error("auth runtime not enabled for the server")
	}

	if tlsConfig != nil {
		// Create the TLS credentials
		cred := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.Creds(cred))
	}

	s := grpc.NewServer(opts...)

	return s, nil
}

// register opencensus metrics views
func (r *runtime) registerMetricsViews() {

	views := DefaultProcessViews
	views = append(views, state.DefaultStoreViews...)
	views = append(views, ocgrpc.DefaultServerViews...)
	views = append(views, ochttp.DefaultServerViews...)

	if err := view.Register(views...); err != nil {
		r.logger.Fatalf("Failed to register ocgrpc server views: %v", err)
	}
}

// register trace exporter
func (r *runtime) registerTraceExporter() {

	if !r.traceEnabled {
		r.logger.Debugf("tracing not enabled")
		return
	}

	var exporter trace.Exporter
	var err error
	switch r.traceBackend {
	case "jaeger":
		// Register the Jaeger exporter to be able to retrieve
		// the collected spans.
		exporter, err = jaeger.NewExporter(jaeger.Options{
			CollectorEndpoint: r.traceEndpoint,
			Process: jaeger.Process{
				ServiceName: r.traceNamespace,
			},
		})
	default:
		r.logger.Warnf("Unsupported tracing backend %s", r.traceBackend)
		return
	}

	if err != nil {
		r.logger.Fatalf("Failed to create an Jaeger Trace exporter %v", err)
		return
	}

	trace.RegisterExporter(exporter)
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
}

// registers prometheus metrics exporter
func (r *runtime) registerPromMetricsExporter(mux *http.ServeMux) {
	// Create the Prometheus exporter.
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
