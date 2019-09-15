package state

import (
	"context"
	"fmt"
	"strings"

	"go.opencensus.io/trace"

	"github.com/cnative/example/pkg/log"
)

// force context to be used
var _ context.Context

// storeWithTrace wraps another Store and records trace information.
type storeWithTrace struct {
	wrappedStore Store
	component    string
}

// StoreWithTrace creates a new Store with trace.
func StoreWithTrace(toWrap Store, logger *log.Logger) Store {
	component := strings.TrimPrefix(fmt.Sprintf("%T", toWrap), "*")
	logger.Debugf("store tracing enabled for %v", component)

	return &storeWithTrace{
		wrappedStore: toWrap,
		component:    component,
	}
}

var _ Store = (*storeWithTrace)(nil)

// Serve .
func (s *storeWithTrace) Serve(ctx context.Context) (r0 error) {
	ctx, span := trace.StartSpan(ctx, "Serve")
	defer span.End()

	r0 = s.wrappedStore.Serve(ctx)

	if r0 != nil {
		span.Annotate([]trace.Attribute{
			trace.StringAttribute("error", r0.Error()),
		}, "Serve")
	}

	return r0
}

// CreateReport .
func (s *storeWithTrace) CreateReport(ctx context.Context, r *Report) (r0 *Report, r1 error) {
	ctx, span := trace.StartSpan(ctx, "CreateReport")
	defer span.End()

	r0, r1 = s.wrappedStore.CreateReport(ctx, r)

	if r1 != nil {
		span.Annotate([]trace.Attribute{
			trace.StringAttribute("error", r1.Error()),
		}, "CreateReport")
	}

	return r0, r1
}

// GetReport .
func (s *storeWithTrace) GetReport(ctx context.Context, id string) (r0 *Report, r1 error) {
	ctx, span := trace.StartSpan(ctx, "GetReport")
	defer span.End()

	r0, r1 = s.wrappedStore.GetReport(ctx, id)

	if r1 != nil {
		span.Annotate([]trace.Attribute{
			trace.StringAttribute("error", r1.Error()),
		}, "GetReport")
	}

	return r0, r1
}

// FilterReports .
func (s *storeWithTrace) FilterReports(ctx context.Context, req FilterRequest) (r0 []*Report, r1 error) {
	ctx, span := trace.StartSpan(ctx, "FilterReports")
	defer span.End()

	r0, r1 = s.wrappedStore.FilterReports(ctx, req)

	if r1 != nil {
		span.Annotate([]trace.Attribute{
			trace.StringAttribute("error", r1.Error()),
		}, "FilterReports")
	}

	return r0, r1
}

// Healthy calls Healthy on the wrapped Store.
func (s *storeWithTrace) Healthy() error {
	return s.wrappedStore.Healthy()
}

// Ready calls Ready on the wrapped Store.
func (s *storeWithTrace) Ready() (bool, error) {
	return s.wrappedStore.Ready()
}

// Close calls Close on the wrapped Store.
func (s *storeWithTrace) Close() error {
	return s.wrappedStore.Close()
}
