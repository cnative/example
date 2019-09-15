package server

import (
	"github.com/cnative/example/pkg/auth"
	"github.com/cnative/example/pkg/log"
	"github.com/cnative/example/pkg/state"
)

type (
	// Option configures choices
	Option interface {
		apply(*runtime)
	}
	optionFunc func(*runtime)
)

// Store used by runtime
func Store(store state.Store) Option {
	return optionFunc(func(r *runtime) {
		r.store = store
	})
}

// Logger for runtime
func Logger(l *log.Logger) Option {
	return optionFunc(func(r *runtime) {
		r.logger = l.NamedLogger("rt")
	})
}

// AuthRuntime sets up AuthN and AuthZ for server runtime
func AuthRuntime(authRuntime auth.Runtime) Option {
	return optionFunc(func(r *runtime) {
		r.authRuntime = authRuntime
	})
}

// Port of the main grpc server
func Port(port uint) Option {
	return optionFunc(func(r *runtime) {
		r.port = port
	})
}

// HealthPort for health check
func HealthPort(port uint) Option {
	return optionFunc(func(r *runtime) {
		r.hPort = port
	})
}

// MetricsPort of the main grpc server
func MetricsPort(port uint) Option {
	return optionFunc(func(r *runtime) {
		r.mPort = port
	})
}

// GatewayPort for HTTP REST/Json End points
func GatewayPort(port uint) Option {
	return optionFunc(func(r *runtime) {
		r.gwPort = port
	})
}

// Gateway enable/disable gateway
func Gateway(enabled bool) Option {
	return optionFunc(func(r *runtime) {
		r.gwEnabled = enabled
	})
}

// Trace enable/disable
func Trace(enabled bool) Option {
	return optionFunc(func(r *runtime) {
		r.traceEnabled = enabled
	})
}

// TraceEP Trace End point
func TraceEP(ep string) Option {
	return optionFunc(func(r *runtime) {
		r.traceEndpoint = ep
	})
}

// TraceBackend jaeger / zipkin
func TraceBackend(backend string) Option {
	return optionFunc(func(r *runtime) {
		r.traceBackend = backend
	})
}

// TraceNamespace used for isolation/categorization
func TraceNamespace(ns string) Option {
	return optionFunc(func(r *runtime) {
		r.traceNamespace = ns
	})
}

// TLSCred Key and Cert Files
func TLSCred(certFile, keyFile, clientCA string) Option {
	return optionFunc(func(r *runtime) {
		r.keyFile = keyFile
		r.certFile = certFile
		r.clientCA = clientCA
	})
}

// APIHandler that needs to be registered with Runtime
func APIHandler(handler GrpcHandler) Option {
	return optionFunc(func(r *runtime) {
		r.apiHandler = handler
	})
}

// ProcessMetrics ebable collection of process metrics
func ProcessMetrics(enabled bool) Option {
	return optionFunc(func(r *runtime) {
		r.processMetricsEnabled = enabled
	})
}

// PProf ebable server profile data
func PProf(enabled bool) Option {
	return optionFunc(func(r *runtime) {
		r.pprofEnabled = enabled
	})
}
