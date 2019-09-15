package state

import (
	"context"

	"go.opencensus.io/stats"

	"github.com/cnative/example/pkg/log"
)

// force context to be used
var _ context.Context

// storeWithMetrics wraps another Store and sends metrics to Prometheus.
type storeWithMetrics struct {
	wrappedStore Store
	observer     *storeObserver
}

// StoreWithMetrics creates a new Store with metrics
func StoreWithMetrics(toWrap Store, logger *log.Logger) Store {
	return &storeWithMetrics{wrappedStore: toWrap, observer: newStoreObserver(logger)}
}

var _ Store = (*storeWithMetrics)(nil)

// Serve .
func (s *storeWithMetrics) Serve(ctx context.Context) (r0 error) {
	done := s.observer.Observe(ctx, "Serve")
	defer done()
	r0 = s.wrappedStore.Serve(ctx)

	if r0 != nil {
		stats.Record(ctx, storeCallErrorCount.M(1)) // Counter to track a wrappedStore call errors
	}

	return r0
}

// CreateReport .
func (s *storeWithMetrics) CreateReport(ctx context.Context, r *Report) (r0 *Report, r1 error) {
	done := s.observer.Observe(ctx, "CreateReport")
	defer done()
	r0, r1 = s.wrappedStore.CreateReport(ctx, r)

	if r1 != nil {
		stats.Record(ctx, storeCallErrorCount.M(1)) // Counter to track a wrappedStore call errors
	}

	return r0, r1
}

// GetReport .
func (s *storeWithMetrics) GetReport(ctx context.Context, id string) (r0 *Report, r1 error) {
	done := s.observer.Observe(ctx, "GetReport")
	defer done()
	r0, r1 = s.wrappedStore.GetReport(ctx, id)

	if r1 != nil {
		stats.Record(ctx, storeCallErrorCount.M(1)) // Counter to track a wrappedStore call errors
	}

	return r0, r1
}

// FilterReports .
func (s *storeWithMetrics) FilterReports(ctx context.Context, req FilterRequest) (r0 []*Report, r1 error) {
	done := s.observer.Observe(ctx, "FilterReports")
	defer done()
	r0, r1 = s.wrappedStore.FilterReports(ctx, req)

	if r1 != nil {
		stats.Record(ctx, storeCallErrorCount.M(1)) // Counter to track a wrappedStore call errors
	}

	return r0, r1
}

// Healthy calls Healthy on the wrapped wrappedStore.
func (s *storeWithMetrics) Healthy() error {
	return s.wrappedStore.Healthy()
}

// Ready calls ready on the wrapped wrappedStore.
func (s *storeWithMetrics) Ready() (bool, error) {
	return s.wrappedStore.Ready()
}

// Close calls Close on the wrapped wrappedStore.
func (s *storeWithMetrics) Close() error {
	return s.wrappedStore.Close()
}
