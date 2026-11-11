package compat_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	eachgen "github.com/sqlc-dev/sqlc/endtoend/wicked_each/go"
	"github.com/stumble/wpgx"
)

// Optional scoped integration check; the standard contract suite needs no DB.
func TestGeneratedEachPostgres(t *testing.T) {
	dsn := os.Getenv("SQLC_EACH_TEST_DSN")
	if dsn == "" {
		t.Skip("set SQLC_EACH_TEST_DSN to a disposable PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := wpgx.NewPool(ctx, &wpgx.Config{
		Username: cfg.User, Password: cfg.Password, Host: cfg.Host, Port: int(cfg.Port), DBName: cfg.Database,
		MaxConns: 1, MinConns: 0, MaxConnLifetime: time.Minute, MaxConnIdleTime: time.Minute, AppName: "sqlc_each_test",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	conn := pool.WConn()
	if _, err := conn.WExec(ctx, "setup", "DROP TABLE IF EXISTS books; CREATE TABLE books (id bigint PRIMARY KEY, value bytea NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	// More than one network buffer remains unread when the visitor first runs.
	if _, err := conn.WExec(ctx, "seed", "INSERT INTO books SELECT n, decode(repeat('61', 131072), 'hex') FROM generate_series(1,64) n"); err != nil {
		t.Fatal(err)
	}
	q := eachgen.New(conn, nil)
	primary := errors.New("policy rejection")
	for _, failVisitor := range []bool{false, true} {
		_, err := pool.Transact(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly}, func(ctx context.Context, tx *wpgx.WTx) (any, error) {
			count := 0
			err := q.WithTx(tx).ListBooksEach(ctx, 0, func(context.Context, eachgen.Book) (bool, error) {
				count++
				if failVisitor {
					return false, primary
				}
				return false, nil
			})
			if count != 1 {
				t.Fatalf("visitor count=%d", count)
			}
			if failVisitor {
				if !errors.Is(err, primary) {
					t.Fatalf("visitor err=%v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			var one int
			if err := tx.WQueryRow(ctx, "probe", "SELECT 1").Scan(&one); err != nil {
				t.Fatalf("drained transaction unusable: %v", err)
			}
			if one != 1 {
				t.Fatalf("probe=%d", one)
			}
			return nil, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	var originalPID int
	_, err = pool.Transact(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly}, func(ctx context.Context, tx *wpgx.WTx) (any, error) {
		if err := tx.WQueryRow(ctx, "pid", "SELECT pg_backend_pid()").Scan(&originalPID); err != nil {
			return nil, err
		}
		readCtx, abort := context.WithCancel(ctx)
		defer abort()
		err := q.WithTx(tx).ListBooksEach(readCtx, 0, func(context.Context, eachgen.Book) (bool, error) {
			abort()
			return false, primary
		})
		if !errors.Is(err, primary) {
			t.Fatalf("Each lost abort's primary error: %v", err)
		}
		return nil, err
	})
	if err == nil {
		t.Fatal("aborted transaction succeeded")
	}
	// Older wpgx may replace the callback error on failed rollback; Each's
	// primary error is asserted inside the callback independently of that API.
	var replacementPID int
	if err := conn.WQueryRow(ctx, "replacement", "SELECT pg_backend_pid()").Scan(&replacementPID); err != nil {
		t.Fatal(err)
	}
	if originalPID == replacementPID {
		t.Fatal("explicitly aborted connection was not replaced")
	}
}
