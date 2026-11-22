package compat_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	eachgen "github.com/sqlc-dev/sqlc/endtoend/wicked_each/go"
	"github.com/stumble/wpgx"
)

// Optional scoped integration check of the generated timeout (1s for every
// wicked_each query): it starts once the query has a connection.
func TestGeneratedTimeoutPostgres(t *testing.T) {
	dsn := os.Getenv("SQLC_EACH_TEST_DSN")
	if dsn == "" {
		t.Skip("set SQLC_EACH_TEST_DSN to a disposable PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	setup, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer setup.Close(ctx)
	schema := pgx.Identifier{fmt.Sprintf("sqlc_timeout_%d", time.Now().UnixNano())}.Sanitize()
	if _, err := setup.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := setup.Exec(cleanup, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("schema cleanup: %v", err)
		}
	}()
	if _, err := setup.Exec(ctx, "CREATE TABLE "+schema+".books (id bigint PRIMARY KEY, value bytea NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, "INSERT INTO "+schema+".books VALUES (1, 'one'), (2, 'two')"); err != nil {
		t.Fatal(err)
	}
	pool, err := wpgx.NewPool(ctx, &wpgx.Config{
		Username: cfg.User, Password: cfg.Password, Host: cfg.Host, Port: int(cfg.Port), DBName: cfg.Database,
		MaxConns: 1, MinConns: 0, MaxConnLifetime: time.Minute, MaxConnIdleTime: time.Minute,
		AppName: "sqlc_timeout_test", AcquireTimeout: 3 * time.Second,
		BeforeAcquire: func(ctx context.Context, conn *pgx.Conn) bool {
			_, err := conn.Exec(ctx, "SET search_path TO "+schema)
			return err == nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	q := eachgen.New(pool.WConn(), nil)

	t.Run("waiting_for_a_connection_does_not_count", func(t *testing.T) {
		held, err := pool.RawPrimaryPool().Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		time.AfterFunc(1300*time.Millisecond, held.Release)
		start := time.Now()
		books, err := q.ListBooks(ctx, 0)
		if err != nil {
			t.Fatalf("a 1.3s wait failed a 1s query: %v", err)
		}
		if len(books) != 2 || time.Since(start) < 1200*time.Millisecond {
			t.Fatalf("books=%d elapsed=%s", len(books), time.Since(start))
		}
	})

	t.Run("visitor_gets_the_statement_deadline", func(t *testing.T) {
		start := time.Now()
		err := q.ListBooksEach(ctx, 0, func(vctx context.Context, _ eachgen.Book) (bool, error) {
			deadline, ok := vctx.Deadline()
			if !ok || deadline.Before(start.Add(900*time.Millisecond)) || deadline.After(time.Now().Add(time.Second)) {
				return false, fmt.Errorf("deadline=%v ok=%v", deadline, ok)
			}
			return true, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("a_visitor_returning_the_deadline_reports_the_database_timeout", func(t *testing.T) {
		err := q.ListBooksEach(ctx, 0, func(vctx context.Context, _ eachgen.Book) (bool, error) {
			<-vctx.Done()
			return false, vctx.Err()
		})
		if !errors.Is(err, wpgx.ErrDBTimeout) || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("draining_past_the_timeout_after_an_early_stop_fails", func(t *testing.T) {
		slow := eachgen.New(&slowCloseConn{WGConn: pool.WConn()}, nil)
		err := slow.ListBooksEach(ctx, 0, func(context.Context, eachgen.Book) (bool, error) {
			return false, nil
		})
		if !errors.Is(err, wpgx.ErrDBTimeout) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("a_blocked_query_reports_the_database_timeout", func(t *testing.T) {
		locker, err := pgx.Connect(ctx, dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer locker.Close(ctx)
		tx, err := locker.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, "LOCK TABLE "+schema+".books IN ACCESS EXCLUSIVE MODE"); err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		_, err = q.ListBooks(ctx, 0)
		if !errors.Is(err, wpgx.ErrDBTimeout) || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("ListBooks err=%v", err)
		}
		err = q.ListBooksEach(ctx, 0, func(context.Context, eachgen.Book) (bool, error) { return true, nil })
		if !errors.Is(err, wpgx.ErrDBTimeout) {
			t.Fatalf("ListBooksEach err=%v", err)
		}
		if elapsed := time.Since(start); elapsed > 4*time.Second {
			t.Fatalf("two 1s timeouts took %s", elapsed)
		}
	})
}

// slowCloseConn returns rows whose Close waits until the statement context
// ends, as a slow drain of buffered rows would.
type slowCloseConn struct{ wpgx.WGConn }

func (c *slowCloseConn) WQuery(ctx context.Context, name, sql string, args ...any) (pgx.Rows, error) {
	rows, err := c.WGConn.WQuery(ctx, name, sql, args...)
	if err != nil {
		return rows, err
	}
	return slowCloseRows{rows}, nil
}

type slowCloseRows struct{ pgx.Rows }

func (r slowCloseRows) Context() context.Context {
	return r.Rows.(interface{ Context() context.Context }).Context()
}

func (r slowCloseRows) Close() {
	<-r.Context().Done()
	r.Rows.Close()
}
