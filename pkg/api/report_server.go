package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/golang/protobuf/ptypes"
	grpc_runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/cnative/example/pkg/log"
	"github.com/cnative/example/pkg/server"
	"github.com/cnative/example/pkg/state"

	"google.golang.org/grpc"
)

type (
	reportsServer struct {
		store  state.Store
		logger *log.Logger
	}
)

var (
	// ErrParsingLabels error while parsing labels string
	ErrParsingLabels = errors.New("parsing of labels failed")
)

// Make sure that streamService implements the api.ReportServiceServer interface
var _ ReportServiceServer = &reportsServer{}

// NewReportsServerHandler Creates a new reports server which implements api.ReportServiceServer
func NewReportsServerHandler(store state.Store, l *log.Logger) (server.GrpcHandler, error) {
	return newReportsServer(store, l)
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
	RegisterReportServiceServer(s, r)
	if mux == nil {
		return nil
	}

	return RegisterReportServiceHandlerServer(ctx, mux, r)
}

func (r *reportsServer) Close() error {
	return nil
}

func (r *reportsServer) CreateReport(ctx context.Context, req *CreateReportRequest) (*CreateReportResponse, error) {

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

	cr := &CreateReportResponse{
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

func (r *reportsServer) FilterReports(ctx context.Context, fr *FilterReportsRequest) (*FilterReportsResponse, error) {

	res, err := r.store.FilterReports(ctx, state.NewFilterReportsRequest(state.Name(fr.Name)))
	if err != nil {
		return nil, err
	}

	frr := []*FilterReportsResponseItem{}
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

		frr = append(frr, &FilterReportsResponseItem{
			Id:        re.ID,
			Name:      re.Name,
			Labels:    labels,
			CreatedBy: re.CreatedBy,
			UpdatedBy: re.UpdatedBy,
			CreatedAt: capb,
			UpdatedAt: uapb,
		})
	}

	return &FilterReportsResponse{
		Reports: frr,
	}, nil
}

func (r *reportsServer) GetReport(ctx context.Context, req *GetReportRequest) (*GetReportResponse, error) {
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

	return &GetReportResponse{
		Id:        re.ID,
		Name:      re.Name,
		Labels:    labels,
		CreatedBy: re.CreatedBy,
		UpdatedBy: re.UpdatedBy,
		CreatedAt: capb,
		UpdatedAt: uapb,
	}, nil
}
