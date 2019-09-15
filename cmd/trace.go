package cmd

import "github.com/urfave/cli"

type (
	//TraceConfig represents config needed for tracing
	TraceConfig struct {
		Enabled   bool
		Endpoint  string
		Namespace string
		Backend   string
	}
)

var (
	// TraceFlags for trace connectivity
	TraceFlags = []cli.Flag{
		cli.BoolFlag{
			Name:   "trace",
			Usage:  "Enable tracing",
			EnvVar: "TRACE_ENABLED",
		},
		cli.StringFlag{
			Name:   "trace-endpoint",
			Value:  "http://localhost:14268/api/traces",
			Usage:  "Endpoint for the trace service",
			EnvVar: "TRACE_ENDPOINT",
		},
		cli.StringFlag{
			Name:   "trace-namespace",
			Usage:  "Service namespace",
			EnvVar: "TRACE_NAMESPACE",
		},
		cli.StringFlag{
			Name:   "trace-backend",
			Usage:  "Backend to use for the tracing",
			Value:  "jaeger",
			EnvVar: "TRACE_BACKEND",
		},
	}
)

// TraceConfigFromCLI returns trace config from the CLI
func TraceConfigFromCLI(c *cli.Context, serviceName string) TraceConfig {
	ns := c.String("trace-namespace")
	if ns == "" {
		ns = serviceName
	}
	return TraceConfig{
		Enabled:   c.Bool("trace"),
		Endpoint:  c.String("trace-endpoint"),
		Namespace: ns,
		Backend:   c.String("trace-backend"),
	}
}
