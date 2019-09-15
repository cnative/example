package main

import (
	"context"
	"fmt"
	olog "log"
	"os"
	"runtime"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"github.com/cnative/example/cmd"

	_ "net/http/pprof"

	_ "github.com/lib/pq"

	"github.com/cnative/example/pkg/api"
	"github.com/cnative/example/pkg/auth"
	"github.com/cnative/example/pkg/log"
	"github.com/cnative/example/pkg/server"
	"github.com/cnative/example/pkg/state"
)

const (
	svcName = "reports-server"
)

var (
	version   = "unknown"
	gitCommit = "unknown"

	app           = cli.NewApp()
	errorExitCode = cli.NewExitError("", 1)

	serverFlags = []cli.Flag{
		cli.UintFlag{
			Name:   "port",
			Value:  2020,
			EnvVar: "SERVER_PORT",
		},
		cli.UintFlag{
			Name:   "gateway-port",
			Value:  2019,
			EnvVar: "GATEWAY_PORT",
		},
		cli.UintFlag{
			Name:   "health-port",
			Value:  2021,
			EnvVar: "HEALTH_PORT",
		},
		cli.UintFlag{
			Name:   "metrics-port",
			Value:  9101,
			EnvVar: "METRICS_PORT",
		},
		cli.BoolFlag{
			Name:   "no-gateway",
			Usage:  "Disable gateway listener",
			EnvVar: "GATEWAY_DISABLED",
		},
		cli.StringFlag{
			Name:  "state-store",
			Usage: "storage driver, currently supported [postgres]",
			Value: "postgres",
		},
		cmd.TLSCertFile,
		cmd.TLSPrivateKeyFile,
		cmd.TLSCertDir,
		cmd.ClientCAFile,
		cmd.InsecureSkipTLS,
		cmd.Debug,
		cmd.SkipProcessMetrics,
		cmd.DBHost,
		cmd.DBPort,
		cmd.DBName,
		cmd.DBUser,
		cmd.DBPassword,
	}
)

type serverConfig struct {
	trace              cmd.TraceConfig
	tls                cmd.TLSConfig
	debug              bool
	port               uint
	hPort              uint
	mPort              uint
	gwPort             uint
	gwEnabled          bool
	stateStore         string
	dbHost             string
	dbPort             uint
	dbUser             string
	dbPassword         string
	dbName             string
	skipProcessMetrics bool
	enablePProf        bool
}

func getRootLogger(debug bool) (*log.Logger, error) {
	ll := log.InfoLevel
	if debug {
		ll = log.DebugLevel
	}

	return log.New(log.WithName(svcName), log.WithLevel(ll))
}

func isPortValid(port uint) bool {
	return port > 0 && port < 65535
}

func getStateStore(logger *log.Logger, o *serverConfig) (store state.Store, err error) {

	switch o.stateStore {
	case "postgres":
		ds := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", o.dbUser, o.dbPassword, o.dbHost, o.dbPort, o.dbName)
		logger.Infow("connecting to database", "datasource", ds)
		store, err = state.NewPostgresStore(logger, ds)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid store: %s", o.stateStore)
	}

	return store, store.Serve(context.Background())
}

func serverConfigFromCLI(c *cli.Context) (*serverConfig, error) {

	port := c.Uint("port")
	if !isPortValid(port) {
		return nil, fmt.Errorf("invalid port number: %d", port)
	}

	gwPort := c.Uint("gateway-port")
	if !isPortValid(gwPort) {
		return nil, fmt.Errorf("invalid gateway port number: %d", gwPort)
	}

	if gwPort == port {
		return nil, errors.New("gw-port and port cannot be the same")
	}

	healthPort := c.Uint("health-port")
	if !isPortValid(healthPort) {
		return nil, fmt.Errorf("invalid health-port number: %d", healthPort)
	}

	if healthPort == port {
		return nil, errors.New("health-port and port cannot be the same")
	}

	metricsPort := c.Uint("metrics-port")
	if !isPortValid(metricsPort) {
		return nil, fmt.Errorf("invalid metrics port number: %d", metricsPort)
	}

	if healthPort == gwPort {
		return nil, errors.New("health-port and gw-port cannot be the same")
	}

	if metricsPort == port {
		return nil, errors.New("metrics-port and port cannot be the same")
	}

	if metricsPort == healthPort {
		return nil, errors.New("metrics-port and health-port cannot be the same")
	}

	if metricsPort == gwPort {
		return nil, errors.New("metrics-port and gw-port cannot be the same")
	}

	tlsConfig, err := cmd.TLSConfigFromCLI(c)
	if err != nil {
		return nil, err
	}

	return &serverConfig{
		trace:              cmd.TraceConfigFromCLI(c, "reports-server"),
		port:               port,
		gwPort:             gwPort,
		hPort:              healthPort,
		mPort:              metricsPort,
		tls:                tlsConfig,
		debug:              c.Bool(cmd.Debug.Name),
		skipProcessMetrics: c.Bool(cmd.SkipProcessMetrics.Name),
		enablePProf:        c.Bool(cmd.PProfEnable.Name),
		stateStore:         c.String("state-store"),
		gwEnabled:          !c.Bool("no-gateway"),
		dbHost:             c.String(cmd.DBHost.Name),
		dbPort:             c.Uint(cmd.DBPort.Name),
		dbUser:             c.String(cmd.DBUser.Name),
		dbPassword:         c.String(cmd.DBPassword.Name),
		dbName:             c.String(cmd.DBName.Name),
	}, nil
}

func reportsServerAction(c *cli.Context) (err error) {
	o, err := serverConfigFromCLI(c)
	if err != nil {
		return errors.Errorf("invalid or missing cli arguments - %v", err)
	}

	logger, err := getRootLogger(o.debug)
	if err != nil {
		return err
	}

	logger.Info("starting server...")

	var authCtx auth.Runtime
	if cliOpts := cmd.OIDCAuthOptionsFromCLI(c); cliOpts != nil {
		authOpts := []auth.Option{
			auth.Logger(logger),
			auth.Authorizer(
				//TODO Implement AuthZ by checking the claims
				func(context.Context, auth.Claims, auth.Resource, auth.Action) (bool, error) {
					return true, nil
				},
			),
		}
		if authCtx, err = auth.NewRuntime(append(authOpts, cliOpts...)...); err != nil {
			logger.Fatalf("unable to create runtime auth context - %v", err)
			return errorExitCode
		}
	}

	store, err := getStateStore(logger, o)
	if err != nil {
		logger.Fatalf("unable to create state store - %v", err)
		return errorExitCode
	}
	defer store.Close()

	handler, err := api.NewReportsServerHandler(store, logger)
	if err != nil {
		logger.Fatalw(fmt.Sprintf("unable to create reports-server handler - %v", err))
		return errorExitCode
	}

	rt, err := server.NewRuntime(
		server.Logger(logger),
		server.AuthRuntime(authCtx),
		server.TLSCred(o.tls.CertFile, o.tls.KeyFile, o.tls.CAFile),
		server.Port(o.port), server.HealthPort(o.hPort), server.MetricsPort(o.mPort),
		server.Store(store),
		server.ProcessMetrics(!o.skipProcessMetrics), server.PProf(o.enablePProf),
		server.Gateway(o.gwEnabled), server.GatewayPort(o.gwPort),
		server.Trace(o.trace.Enabled), server.TraceBackend(o.trace.Backend), server.TraceNamespace(o.trace.Namespace), server.TraceEP(o.trace.Endpoint),
		server.APIHandler(handler),
	)
	if err != nil {
		logger.Errorf("unable to create server runtime - %v", err)
		return errorExitCode
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errc, err := rt.Start(ctx)
	if err != nil {
		return errorExitCode
	}

	logger.Info("server started")

	err = <-errc // blocking on error channel
	if err != nil {
		logger.Errorf("Received error from error channel %v", err)
	}
	rt.Stop(context.Background())

	return err
}

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("%s\n Version:  %s\n Git Commit:  %s\n Go Version:  %s\n OS/Arch:  %s/%s\n Built:  %s\n",
			svcName, version, gitCommit, runtime.Version(), runtime.GOOS, runtime.GOARCH, c.App.Compiled)
	}

	app.Name = svcName
	app.Copyright = "(c) 2019 Copyright"
	app.Usage = "reports server"

	app.Version = version
	app.Flags = append(serverFlags, append(cmd.OIDCFlags, cmd.TraceFlags...)...)
	app.Action = reportsServerAction

	if err := app.Run(os.Args); err != nil {
		olog.SetFlags(0)
		olog.Fatalf("%v\n", err)
	}
}
