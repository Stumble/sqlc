package compat_test

// Keep this cross-package contract suite in the existing wicked CI entrypoint.
// It executes the generated Each fixture rather than reimplementing its loop.
import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	eachgen "github.com/sqlc-dev/sqlc/endtoend/wicked_each/go"
	"github.com/stumble/dcache"
)

type eachRows struct {
	data                                   []eachgen.Book
	index                                  int
	closed                                 bool
	closeCanceled                          bool
	ctx                                    context.Context
	scanErr, sourceErr, closeErr, finalErr error
	failAfter                              int
}

func (r *eachRows) Close() {
	if r.closed {
		return
	}
	r.closeCanceled = r.ctx.Err() != nil
	r.closed = true
	if r.finalErr == nil {
		r.finalErr = r.closeErr
	}
	if r.finalErr == nil {
		r.finalErr = r.ctx.Err()
	}
}
func (r *eachRows) Err() error { return r.finalErr }
func (r *eachRows) Next() bool {
	if r.closed {
		return false
	}
	if r.sourceErr != nil && r.index == r.failAfter {
		r.finalErr = r.sourceErr
		r.Close()
		return false
	}
	if r.index == len(r.data) {
		r.Close()
		return false
	}
	r.index++
	return true
}
func (r *eachRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := r.data[r.index-1]
	*dest[0].(*int64) = row.ID
	if len(dest) > 1 {
		*dest[1].(*[]byte) = append([]byte(nil), row.Value...)
	}
	return nil
}
func (*eachRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (*eachRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (*eachRows) Values() ([]any, error)                       { panic("unexpected Values") }
func (*eachRows) RawValues() [][]byte                          { return nil }
func (*eachRows) Conn() *pgx.Conn                              { return nil }
func (*eachRows) TypeMap() *pgtype.Map                         { return nil }

type eachConn struct {
	rows     *eachRows
	calls    int
	intents  []string
	name     string
	args     []any
	queryErr error
}

func (c *eachConn) WQuery(ctx context.Context, name, sql string, args ...any) (pgx.Rows, error) {
	c.calls++
	c.name, c.args = name, args
	c.rows.ctx = ctx
	if c.queryErr != nil {
		return nil, c.queryErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c.rows, nil
}
func (*eachConn) WQueryRow(context.Context, string, string, ...any) pgx.Row {
	panic("unexpected WQueryRow")
}
func (c *eachConn) CountIntent(name string) { c.intents = append(c.intents, name) }
func (*eachConn) WExec(context.Context, string, string, ...any) (pgconn.CommandTag, error) {
	panic("unexpected WExec")
}
func (*eachConn) WCopyFrom(context.Context, string, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	panic("unexpected WCopyFrom")
}
func (*eachConn) PostExec(func() error) error { panic("unexpected PostExec") }

func newEachConn() *eachConn {
	return &eachConn{rows: &eachRows{data: []eachgen.Book{
		{ID: 1, Value: []byte("one")},
		{ID: 2, Value: []byte("two")},
		{ID: 3, Value: []byte("three")},
	}}}
}

func TestGeneratedEachVisitsAndOwnsLifecycle(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		t.Run(map[bool]string{false: "queries", true: "readonly"}[readOnly], func(t *testing.T) {
			conn := newEachConn()
			q := eachgen.New(conn, nil)
			var got []eachgen.Book
			var visitCtx context.Context
			visit := func(ctx context.Context, row eachgen.Book) (bool, error) {
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("visitor lacks query deadline")
				}
				visitCtx = ctx
				got = append(got, row)
				return true, nil
			}
			var err error
			if readOnly {
				err = q.AsReadOnly().ListBooksEach(context.Background(), 1, visit)
			} else {
				err = q.ListBooksEach(context.Background(), 1, visit)
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, conn.rows.data) {
				t.Fatalf("rows=%v", got)
			}
			if !conn.rows.closed || conn.rows.closeCanceled {
				t.Fatal("rows were not drained before cancellation")
			}
			if visitCtx.Err() != context.Canceled {
				t.Fatal("query context outlived Each")
			}
			if conn.calls != 1 || !reflect.DeepEqual(conn.intents, []string{"eachgen.ListBooks"}) || !reflect.DeepEqual(conn.args, []any{int64(1)}) {
				t.Fatalf("calls=%d intents=%v args=%v", conn.calls, conn.intents, conn.args)
			}
			got[0].Value[0] = 'X'
			if string(got[1].Value) != "two" {
				t.Fatal("retained rows alias")
			}
		})
	}
}

func TestGeneratedEachStopAndErrors(t *testing.T) {
	primary := errors.New("primary")
	cleanup := errors.New("close-only failure")
	for _, tc := range []struct {
		name                                               string
		stop                                               bool
		visitorErr, scanErr, sourceErr, closeErr, queryErr error
		want                                               error
		visits                                             int
	}{
		{name: "stop", stop: true, visits: 1},
		{name: "stop_close_error", stop: true, closeErr: cleanup, want: cleanup, visits: 1},
		{name: "visitor_error", visitorErr: primary, want: primary, visits: 1},
		{name: "visitor_wins_over_cleanup", visitorErr: primary, closeErr: cleanup, want: primary, visits: 1},
		{name: "scan_error", scanErr: primary, closeErr: cleanup, want: primary},
		{name: "source_error", sourceErr: primary, want: primary, visits: 1},
		{name: "end_close_error", closeErr: cleanup, want: cleanup, visits: 3},
		{name: "query_error", queryErr: primary, want: primary},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := newEachConn()
			conn.rows.scanErr, conn.rows.sourceErr, conn.rows.closeErr = tc.scanErr, tc.sourceErr, tc.closeErr
			conn.rows.failAfter = 1
			conn.queryErr = tc.queryErr
			visits := 0
			err := eachgen.New(conn, nil).ListBooksEach(context.Background(), 0, func(context.Context, eachgen.Book) (bool, error) {
				visits++
				return !tc.stop, tc.visitorErr
			})
			if !errors.Is(err, tc.want) || visits != tc.visits {
				t.Fatalf("err=%v visits=%d, want %v/%d", err, visits, tc.want, tc.visits)
			}
			if tc.queryErr == nil && (!conn.rows.closed || conn.rows.closeCanceled) {
				t.Fatal("ordinary errors must drain before own cancellation")
			}
		})
	}
}

func TestGeneratedEachNilAndEmpty(t *testing.T) {
	conn := newEachConn()
	q := eachgen.New(conn, nil)
	if err := q.ListBooksEach(context.Background(), 0, nil); err == nil {
		t.Fatal("nil visitor accepted")
	}
	if conn.calls != 0 || len(conn.intents) != 0 {
		t.Fatal("nil visitor reached connection")
	}
	conn.rows.data = nil
	err := q.ListBooksEach(context.Background(), 0, func(context.Context, eachgen.Book) (bool, error) {
		t.Fatal("empty result called visitor")
		return false, nil
	})
	if err != nil || !conn.rows.closed {
		t.Fatalf("empty err=%v closed=%v", err, conn.rows.closed)
	}
}

func TestGeneratedEachCancellation(t *testing.T) {
	primary := errors.New("read policy rejected row")
	for _, tc := range []struct {
		name                               string
		preCancel, deadline, returnPrimary bool
	}{
		{name: "before_query", preCancel: true},
		{name: "deadline_in_visitor", deadline: true},
		{name: "explicit_abort", returnPrimary: true},
		{name: "cancel_normal_stop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ctx context.Context
			var cancel context.CancelFunc
			if tc.deadline {
				ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
			} else {
				ctx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()
			if tc.preCancel {
				cancel()
			}
			conn := newEachConn()
			visits := 0
			err := eachgen.New(conn, nil).ListBooksEach(ctx, 0, func(ctx context.Context, row eachgen.Book) (bool, error) {
				visits++
				if tc.deadline {
					<-ctx.Done()
				} else {
					cancel()
				}
				if tc.returnPrimary {
					return false, primary
				}
				return false, nil
			})
			want := error(context.Canceled)
			if tc.deadline {
				want = context.DeadlineExceeded
			}
			if tc.returnPrimary {
				want = primary
			}
			if !errors.Is(err, want) {
				t.Fatalf("err=%v want=%v", err, want)
			}
			if tc.preCancel {
				if visits != 0 {
					t.Fatal("cancelled query called visitor")
				}
			} else if !conn.rows.closed || !conn.rows.closeCanceled {
				t.Fatal("explicit abort/deadline must close under cancellation")
			}
		})
	}
}

func TestGeneratedEachPanic(t *testing.T) {
	conn := newEachConn()
	marker := &struct{}{}
	defer func() {
		if recover() != marker {
			t.Fatal("visitor panic not propagated")
		}
		if !conn.rows.closed || conn.rows.closeCanceled || conn.rows.ctx.Err() != context.Canceled {
			t.Fatal("panic cleanup/lifetime order wrong")
		}
	}()
	_ = eachgen.New(conn, nil).ListBooksEach(context.Background(), 0, func(context.Context, eachgen.Book) (bool, error) { panic(marker) })
}

func TestGeneratedEachScalarParamsReplicaAndCache(t *testing.T) {
	for _, scalar := range []bool{false, true} {
		conn := newEachConn()
		q := eachgen.New(conn, nil)
		var got []int64
		visit := func(_ context.Context, id int64) (bool, error) { got = append(got, id); return true, nil }
		var err error
		if scalar {
			err = q.ListIDsEach(context.Background(), visit)
		} else {
			err = q.ListNamedScalarEach(context.Background(), visit)
		}
		if err != nil || !reflect.DeepEqual(got, []int64{1, 2, 3}) {
			t.Fatalf("scalar=%v err=%v rows=%v", scalar, err, got)
		}
	}
	conn := newEachConn()
	// An unconfigured cache would fail if touched. Each must bypass it entirely.
	q := eachgen.New(conn, new(dcache.DCache))
	err := q.ListCachedBooksEach(context.Background(), func(context.Context, eachgen.Book) (bool, error) { return false, nil })
	if err != nil || conn.calls != 1 {
		t.Fatalf("cache bypass err=%v calls=%d", err, conn.calls)
	}
	conn = newEachConn()
	q = eachgen.New(conn, nil)
	if err := q.ListPrimaryBooksEach(context.Background(), func(context.Context, eachgen.Book) (bool, error) { return false, nil }); err != nil {
		t.Fatal(err)
	}
	if len(conn.intents) != 0 {
		t.Fatal("disabled CountIntent called")
	}
	if _, ok := reflect.TypeOf(q.AsReadOnly()).MethodByName("ListPrimaryBooksEach"); ok {
		t.Fatal("primary-only method exposed on readonly receiver")
	}
	conn = newEachConn()
	q = eachgen.New(conn, nil)
	if err := q.ListRangeEach(context.Background(), eachgen.ListRangeParams{FromID: 1, ToID: 3}, func(context.Context, eachgen.Book) (bool, error) { return false, nil }); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(conn.args, []any{int64(1), int64(3)}) {
		t.Fatalf("Params args=%v", conn.args)
	}
	replica := newEachConn()
	if err := q.UseReplica(replica).ListBooksEach(context.Background(), 0, func(context.Context, eachgen.Book) (bool, error) { return false, nil }); err != nil {
		t.Fatal(err)
	}
	if replica.calls != 1 || conn.calls != 1 {
		t.Fatal("replica routing changed")
	}
}
