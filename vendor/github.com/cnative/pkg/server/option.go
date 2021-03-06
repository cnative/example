package server

import (
	"fmt"
	"net/http"

	"go.opencensus.io/stats/view"

	"github.com/cnative/pkg/log"

	"github.com/cnative/pkg/health"

	"github.com/cnative/pkg/auth"
)

type (
	// Option configures choices
	Option interface {
		apply(*runtime)
	}
	optionFunc func(*runtime)
)

// Probes used by runtime to check health
func Probes(probes map[string]health.Probe) Option {
	return optionFunc(func(r *runtime) {
		r.probes = probes
	})
}

// Logger for runtime
func Logger(l *log.Logger) Option {
	return optionFunc(func(r *runtime) {
		r.logger = l.NamedLogger("rt")
	})
}

// Debug enables debug and sets up port for http/pprof data
func Debug(debug bool, port uint) Option {
	return optionFunc(func(r *runtime) {
		r.debugEnabled = debug
		r.dPort = port
	})
}

// Tags name value label pairs that is applied to server for info purpuse
func Tags(tags map[string]string) Option {
	return optionFunc(func(r *runtime) {
		r.tags = tags
	})
}

// AuthRuntime sets up AuthN and AuthZ for server runtime
func AuthRuntime(authRuntime auth.Runtime) Option {
	return optionFunc(func(r *runtime) {
		r.authRuntime = authRuntime
	})
}

// GRPCPort of the main grpc server
func GRPCPort(port uint) Option {
	return optionFunc(func(r *runtime) {
		r.gPort = port
	})
}

// HTTPPort of the main http server
func HTTPPort(port uint) Option {
	return optionFunc(func(r *runtime) {
		r.htPort = port
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

// Gateway enable/disable gateway
func Gateway(enabled bool) Option {
	return optionFunc(func(r *runtime) {
		r.gwEnabled = enabled
	})
}

// GatewayPort grpc gateway listener port
func GatewayPort(port uint) Option {
	return optionFunc(func(r *runtime) {
		r.gwPort = port
	})
}

// Trace enable/disable
func Trace(enabled bool) Option {
	return optionFunc(func(r *runtime) {
		r.traceEnabled = enabled
	})
}

// OCAgentEP Opencensus Agent End point
func OCAgentEP(host string, port uint) Option {
	return optionFunc(func(r *runtime) {
		r.ocAgentEP = fmt.Sprintf("%s:%d", host, port)
	})
}

// OCAgentNamespace used for isolation/categorization
func OCAgentNamespace(ns string) Option {
	return optionFunc(func(r *runtime) {
		r.ocAgentNamespace = ns
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

// GRPCAPI that needs to be registered with Runtime
func GRPCAPI(handler GRPCAPIHandler) Option {
	return optionFunc(func(r *runtime) {
		r.grpcAPIHandler = handler
		r.grpcEnabled = true
	})
}

// HTTPAPI that needs to be registered with Runtime
func HTTPAPI(handler http.Handler) Option {
	return optionFunc(func(r *runtime) {
		r.httpHandler = handler
		r.htEnabled = true
	})
}

// CustomMetricsViews custom metrics
func CustomMetricsViews(views ...*view.View) Option {
	return optionFunc(func(r *runtime) {
		r.statsViews = views
	})
}

// ProcessMetrics ebable collection of process metrics
func ProcessMetrics(enabled bool) Option {
	return optionFunc(func(r *runtime) {
		r.processMetricsEnabled = enabled
	})
}
