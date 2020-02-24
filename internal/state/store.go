package state

import (
	"context"
	"errors"
	"io"

	"github.com/cnative/pkg/health"
)

//go:generate servicebuilder iwrap -z -f ./store.go -i Store --output-dir ./ -p state -m "github.com/cnative/pkg/log"

const (
	// ASC Ascending sort order
	ASC SortOrder = iota - 1
	// DESC is Descending sort order
	DESC
)

var (
	// ErrNotImplemented not implmented yet
	ErrNotImplemented = errors.New("not implemented")

	// DefaultPageSize is the number of rows returned by default
	DefaultPageSize = 25
)

//SortOrder indicate Sort Order
type SortOrder int8

// Labels is a name value pairs that can be applied to a resource
type Labels map[string]string

// Store provides access to data that is required for .
type Store interface {
	Serve(ctx context.Context) error
	io.Closer
	health.Probe

	CreateReport(ctx context.Context, r *Report) (*Report, error)
	GetReport(ctx context.Context, id string) (*Report, error)
	FilterReports(ctx context.Context, req FilterRequest) ([]*Report, error)
}

// FilterRequest used for listing
type FilterRequest interface {
	Name() string
	SortingOrder() SortOrder
	SortBy() []string
	Page() int32
	PageSize() int32
}

type filterRequest struct {
	name      string
	sortBy    []string
	sortOrder SortOrder
	pageSize  int32
	page      int32
}

// FilterOption used for listing
type FilterOption interface {
	apply(*filterRequest)
}
type filterOptionFunc func(*filterRequest)

func (f filterOptionFunc) apply(s *filterRequest) {
	f(s)
}

// FilterReportsRequest to filter reports
type FilterReportsRequest interface {
	FilterRequest
}

// NewFilterReportsRequest used for searching
func NewFilterReportsRequest(opts ...FilterOption) FilterReportsRequest {
	//setup defaults
	lReq := &filterRequest{
		sortBy:    []string{"name"},
		sortOrder: ASC,
		page:      1,
		pageSize:  int32(DefaultPageSize),
	}

	for _, opt := range opts {
		opt.apply(lReq)
	}

	return lReq
}

func (l *filterRequest) Name() string {

	return l.name
}

func (l *filterRequest) SortingOrder() SortOrder {

	return l.sortOrder
}

func (l *filterRequest) SortBy() []string {

	return l.sortBy
}

func (l *filterRequest) Page() int32 {
	return l.page
}

func (l *filterRequest) PageSize() int32 {
	return l.pageSize
}

// Name to filter by Name
func Name(name string) FilterOption {
	return filterOptionFunc(func(l *filterRequest) {
		l.name = name
	})
}

// SortBy option to set the SortBy columns
func SortBy(cols ...string) FilterOption {
	return filterOptionFunc(func(l *filterRequest) {
		sortCols := append([]string{}, cols...)

		l.sortBy = sortCols
	})
}

// SortingOrder option use. ASC / DESC
func SortingOrder(order SortOrder) FilterOption {
	return filterOptionFunc(func(l *filterRequest) {
		l.sortOrder = order
	})
}

// Page option to set the page number
func Page(page int32) FilterOption {
	return filterOptionFunc(func(l *filterRequest) {
		l.page = page
		if page < 1 {
			l.page = 1
		}
	})
}

// PageSize option to set the page number
func PageSize(pageSize int32) FilterOption {
	return filterOptionFunc(func(l *filterRequest) {
		l.pageSize = pageSize
		if pageSize < 1 || pageSize > int32(DefaultPageSize) {
			l.pageSize = int32(DefaultPageSize)
		}
	})
}
