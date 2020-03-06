package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/ptypes"
	grpc_runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cnative/pkg/auth"
	"github.com/cnative/pkg/health"
	"github.com/cnative/pkg/log"
	"github.com/cnative/pkg/server"

	"github.com/cnative/example/internal/state"
	"github.com/cnative/example/pkg/api"

	"google.golang.org/grpc"
)

type (
	reportsServer struct {
		store  state.Store
		logger *log.Logger
	}
)

// Make sure that streamService implements the api.ReportServiceServer interface
var _ api.ReportServiceServer = &reportsServer{}

var (
	dbCfg dbConfig

	// ErrParsingLabels error while parsing labels string
	ErrParsingLabels = errors.New("parsing of labels failed")
)

// reportsServerCmd represents api server
var reportsServerCmd = &cobra.Command{
	Use:     "reports",
	Short:   "reports api server. provides grpc and rest api endpoints",
	Aliases: []string{"api-server", "cp"},

	PreRun: func(cmd *cobra.Command, args []string) {
		applyReportsServerCLIArgs(&srvCfg, &dbCfg)
	},

	RunE: startReportsServerServer,
}

func init() {
	rootCmd.AddCommand(reportsServerCmd)

	// reportsserver supports both grpc and gtrpc gateway (http) ports
	reportsServerCmd.PersistentFlags().Int16("grpc-port", reportsServerGRPCPort, "grpc port")
	reportsServerCmd.PersistentFlags().Int16("gateway-port", reportsServerGRPGatewayPort, "grpc gateway port")
	reportsServerCmd.PersistentFlags().BoolP("no-gateway", "", false, "start server with no gateway")

	applyDBFlags(reportsServerCmd, "reports") // Database connect flags
}

func applyReportsServerCLIArgs(sc *serverConfig, dc *dbConfig) {

	sc.gPort = uint(viper.GetInt("grpc-port"))
	sc.gwPort = uint(viper.GetInt("gateway-port"))
	sc.gwEnabled = !viper.GetBool("no-gateway")

	applyDBConfigs(dc, "reports")
}

func startReportsServerServer(cmd *cobra.Command, args []string) error {

	serviceName := "reports"

	logger, err := getRootLogger(srvCfg.debug, serviceName)
	if err != nil {
		return err
	}

	logger.Infof("starting oidc %s server", serviceName)

	authOpts := []auth.Option{
		auth.Logger(logger),
		auth.Authorizer(
			// only supports Authentication and does not support Authorization.
			// To use the end user MUST be authenticated and present a Bearer Token.

			// TODO Eventual goal is to define authorization policy
			// and use the claim in the token to allow for authorization of each
			// request.
			//
			// When implemented, this can be customized per-service
			func(context.Context, auth.Claims, auth.Resource, auth.Action) bool {
				return true
			},
		),
		// auth.AdditionalClaimsProvider(func() interface{} {
		// 	return &rid_auth.BaseClaims{}
		// }),
		// auth.IDResolver(func(cl auth.Claims) string {
		// 	if ac := cl.GetAdditionalClaims(); ac != nil {
		// 		if ridClaims, ok := ac.(*rid_auth.BaseClaims); ok {
		// 			return ridClaims.GetConnectorUserID()
		// 		}
		// 	}
		// 	return ""
		// }),
	}

	// basic options that are common for all servers
	opts := []server.Option{
		server.Logger(logger),
		server.Debug(srvCfg.debug, srvCfg.dPort), server.HealthPort(srvCfg.hPort), server.MetricsPort(srvCfg.mPort),
		server.ProcessMetrics(!srvCfg.skipProcessMetrics), server.Tags(srvCfg.tags),
		server.Trace(srvCfg.ocAgent.traceEnabled), server.OCAgentEP(srvCfg.ocAgent.host, srvCfg.ocAgent.port), server.OCAgentNamespace(srvCfg.ocAgent.namespace),
	}

	if !srvCfg.tls.skip {
		opts = append(opts, server.TLSCred(srvCfg.tls.certFile, srvCfg.tls.keyFile, srvCfg.tls.caFile))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// if the server needs open id connect based auth then
	if oidcOpts := srvCfg.auth.asRuntimeAuthOptions(); len(oidcOpts) > 0 {
		//service must setup necesary command line args for this
		authCtx, err := auth.NewRuntime(ctx, append(authOpts, oidcOpts...)...)
		if err != nil {
			return errors.Wrapf(err, "unable to create oidc runtime auth context")
		}
		opts = append(opts, server.AuthRuntime(authCtx))
	}

	store, err := getStateStore(ctx, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to create %s store", serviceName)
	}
	defer store.Close()
	if handler, err := newReportsServer(store, logger); err == nil {
		opts = append(opts,
			server.Probes(map[string]health.Probe{"store": store}),
			server.CustomMetricsViews(state.DefaultStoreViews...),
			server.GRPCAPI(handler), server.GRPCPort(srvCfg.gPort),
			server.Gateway(srvCfg.gwEnabled), server.GatewayPort(srvCfg.gwPort),
		)
	} else {
		return errors.Wrapf(err, "unable to create %s handler", serviceName)
	}

	rt, err := server.NewRuntime(ctx, serviceName, opts...)
	if err != nil {
		return errors.Wrapf(err, "unable to create %s server runtime", serviceName)
	}

	errc, err := rt.Start(ctx)
	if err != nil {
		return err
	}

	logger.Infof("starting %s server...", serviceName)

	err = <-errc // blocking on error channel
	if err != nil {
		logger.Errorf("Received error from error channel %v", err)
	}
	rt.Stop(ctx)

	return err
}

func newReportsServer(store state.Store, l *log.Logger) (*reportsServer, error) {
	if l == nil {
		l, _ = log.NewNop()
	}

	return &reportsServer{
		store:  store,
		logger: l,
	}, nil
}

// Register registers grpc Server and the grpc Gateway Impl
func (r *reportsServer) Register(ctx context.Context, s *grpc.Server, mux *grpc_runtime.ServeMux) error {
	api.RegisterReportServiceServer(s, r)
	if mux == nil {
		return nil
	}

	return api.RegisterReportServiceHandlerServer(ctx, mux, r)
}

func (r *reportsServer) Close() error {
	return nil
}

func (r *reportsServer) CreateReport(ctx context.Context, req *api.CreateReportRequest) (*api.CreateReportResponse, error) {

	jsonString, err := json.Marshal(req.Labels)
	if err != nil {
		return nil, ErrParsingLabels
	}
	re, err := r.store.CreateReport(ctx, &state.Report{
		Name:   req.Name,
		Labels: string(jsonString),
	})
	if err != nil {
		return nil, err
	}

	labels := make(map[string]string)

	if err := json.Unmarshal([]byte(re.Labels), &labels); err != nil {
		return nil, err
	}

	cr := &api.CreateReportResponse{
		Id:        re.ID,
		Name:      re.Name,
		Labels:    labels,
		CreatedBy: re.CreatedBy,
		UpdatedBy: re.UpdatedBy,
	}

	capb, err := ptypes.TimestampProto(re.CreatedAt)
	if err != nil {
		return nil, err
	}
	cr.CreatedAt = capb

	uapb, err := ptypes.TimestampProto(re.UpdatedAt)
	if err != nil {
		return nil, err
	}
	cr.UpdatedAt = uapb

	return cr, nil
}

func (r *reportsServer) FilterReports(ctx context.Context, fr *api.FilterReportsRequest) (*api.FilterReportsResponse, error) {

	res, err := r.store.FilterReports(ctx, state.NewFilterReportsRequest(state.Name(fr.Name)))
	if err != nil {
		return nil, err
	}

	frr := []*api.FilterReportsResponseItem{}
	for _, re := range res {
		capb, err := ptypes.TimestampProto(re.CreatedAt)
		if err != nil {
			return nil, err
		}

		uapb, err := ptypes.TimestampProto(re.UpdatedAt)
		if err != nil {
			return nil, err
		}
		labels := make(map[string]string)

		if err := json.Unmarshal([]byte(re.Labels), &labels); err != nil {
			return nil, err
		}

		frr = append(frr, &api.FilterReportsResponseItem{
			Id:        re.ID,
			Name:      re.Name,
			Labels:    labels,
			CreatedBy: re.CreatedBy,
			UpdatedBy: re.UpdatedBy,
			CreatedAt: capb,
			UpdatedAt: uapb,
		})
	}

	return &api.FilterReportsResponse{
		Reports: frr,
	}, nil
}

func (r *reportsServer) GetReport(ctx context.Context, req *api.GetReportRequest) (*api.GetReportResponse, error) {
	re, err := r.store.GetReport(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	capb, err := ptypes.TimestampProto(re.CreatedAt)
	if err != nil {
		return nil, err
	}

	uapb, err := ptypes.TimestampProto(re.UpdatedAt)
	if err != nil {
		return nil, err
	}
	labels := make(map[string]string)

	if err := json.Unmarshal([]byte(re.Labels), &labels); err != nil {
		return nil, err
	}

	return &api.GetReportResponse{
		Id:        re.ID,
		Name:      re.Name,
		Labels:    labels,
		CreatedBy: re.CreatedBy,
		UpdatedBy: re.UpdatedBy,
		CreatedAt: capb,
		UpdatedAt: uapb,
	}, nil
}

func getStateStore(ctx context.Context, logger *log.Logger) (store state.Store, err error) {

	ds := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=require", dbCfg.user, dbCfg.password, dbCfg.host, dbCfg.port, dbCfg.name)
	logger.Infow("connecting to database", "datasource", ds)
	store, err = state.NewPostgresStore(logger, ds)
	if err != nil {
		return nil, err
	}

	return store, store.Serve(ctx)
}
