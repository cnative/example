package state

import (
	"context"

	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang-migrate/migrate"
	bindata "github.com/golang-migrate/migrate/source/go_bindata"
	"github.com/google/uuid"

	_ "github.com/golang-migrate/migrate/database/postgres"

	"github.com/cnative/pkg/auth"
	"github.com/cnative/pkg/log"

	"github.com/cnative/example/db/postgres/migrations"
)

var (
	// ErrMissingReportName missing report name
	ErrMissingReportName = errors.New("missing report name")
)

type sqlStore struct {
	db         *sqlx.DB
	logger     *log.Logger
	dataSource string
}

type structScanner interface {
	StructScan(dest interface{}) error
}

func scanReport(scanner structScanner) (*Report, error) {
	var report Report
	if err := scanner.StructScan(&report); err != nil {
		return nil, err
	}
	return &report, nil
}

// NewPostgresStore returns a postgres sql store
func NewPostgresStore(logger *log.Logger, ds string) (Store, error) {

	db, err := sqlx.Connect("postgres", ds)
	if err != nil {
		return nil, err
	}

	return &sqlStore{db: db, logger: logger.NamedLogger("db"), dataSource: ds}, nil
}

func (s *sqlStore) Serve(ctx context.Context) error {
	s.logger.Info("performing db migrations..")
	rs := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			s.logger.Debugf("applying... %v", name)
			return migrations.Asset(name)
		})

	d, err := bindata.WithInstance(rs)
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("migrations", d, s.dataSource)
	if err != nil {
		return err
	}

	if err = m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			s.logger.Debug("no migrations to apply")
		} else {
			return err
		}
	}

	s.logger.Debug("migration completed. closing ..")
	if serr, derr := m.Close(); serr != nil || derr != nil {
		return errors.Errorf("source close err=%v database close err=%v", serr, derr)
	}

	return nil
}

func (s *sqlStore) Close() error {

	return s.db.Close()
}

func (s *sqlStore) Healthy() error {

	return s.db.Ping()
}

func (s *sqlStore) Ready() (bool, error) {

	if err := s.db.Ping(); err != nil {
		return false, err
	}

	return true, nil
}

func (s *sqlStore) CreateReport(ctx context.Context, r *Report) (*Report, error) {

	if r.Name == "" {
		return nil, ErrMissingReportName
	}
	// validate labels

	r.ID = uuid.New().String()
	r.CreatedBy = auth.CurrentUser(ctx)
	r.UpdatedBy = r.CreatedBy

	if err := s.process(ctx, r, s.create); err != nil {
		return nil, err
	}

	return r, nil
}

func (s *sqlStore) create(ctx context.Context, tx *sqlx.Tx, r *Report) error {
	const queryCreate = "insert into reports (id, name, labels, created_by, updated_by, created_at, updated_at) values (:id, :name, :labels, :created_by, :updated_by, now(), now()) returning created_at, updated_at"
	return namedQueryAndScan(ctx, tx, queryCreate, r)
}

func (s *sqlStore) GetReport(ctx context.Context, id string) (*Report, error) {

	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "missing id")
	}

	const queryGet = "select id, name, labels, created_by, created_at, updated_by, updated_at from reports where id=$1"
	row := s.db.QueryRowxContext(ctx, queryGet, id)
	report, err := scanReport(row)
	if err != nil {
		if err == sql.ErrNoRows {
			err = status.Error(codes.NotFound, "report with that id not found")
		}
		return nil, err
	}

	return report, nil
}

func (s *sqlStore) FilterReports(ctx context.Context, fr FilterRequest) ([]*Report, error) {

	const (
		queryListAll    = "select id, name, labels, created_by, created_at, updated_by, updated_at from reports"
		queryListByName = "select id, name, labels, created_by, created_at, updated_by, updated_at from reports where name like $1"
	)

	var (
		rows *sqlx.Rows
		err  error
	)

	if fr.Name() == "" {
		rows, err = s.db.QueryxContext(ctx, queryListAll)
	} else {
		rows, err = s.db.QueryxContext(ctx, queryListByName, "%"+fr.Name()+"%")
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := []*Report{}

	for rows.Next() {

		re, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, re)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return reports, nil
}

// process begins a database transaction, then calls each fn in fns in turn, passing
// in dbReport to each. If a function errors, the transaction is rolled back and its error
// is returned. If all functions are succesful, the transaction is committed and the
// returned error is from tx.Commit.
func (s *sqlStore) process(ctx context.Context, dbReport *Report, fns ...func(ctx context.Context, tx *sqlx.Tx, dbReport *Report) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // does nothing if the Commit below is successful

	for _, fn := range fns {
		if err := fn(ctx, tx, dbReport); err != nil {
			if err == sql.ErrNoRows {
				err = status.Error(codes.NotFound, "report with that id not found")
			}
			return err
		}
	}

	return tx.Commit()
}

// namedQueryAndScan is a combination of sqlx's NamedQueryContext and
// QueryRowxContext, using StructScan to scan the result into dbReport.
func namedQueryAndScan(ctx context.Context, tx *sqlx.Tx, query string, dbReport *Report) error {
	rows, err := sqlx.NamedQueryContext(ctx, tx, query, dbReport)
	if err != nil {
		return err
	}
	defer rows.Close()

	// The query is expected to return a single row so if there is not a first
	// row something has gone wrong.
	if !rows.Next() {
		err := rows.Err()
		if err == nil {
			err = sql.ErrNoRows
		}
		return err
	}

	if err := rows.StructScan(&dbReport); err != nil {
		return err
	}

	return rows.Close()
}
