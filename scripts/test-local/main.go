// test-local runs a command against PostgreSQL and MySQL containers owned by
// this invocation. It never discovers or reuses a database on the host.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	// Readiness attempts can encounter a restarting initialization server.
	// Report the final readiness error instead of logging every retry.
	if err := mysql.SetLogger(log.New(io.Discard, "", 0)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: test-local -- command [args...]")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var ids []string
	defer func() {
		for i := len(ids) - 1; i >= 0; i-- {
			cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			out, err := exec.CommandContext(cleanup, "docker", "rm", "--force", ids[i]).CombinedOutput()
			cancel()
			if err != nil {
				fmt.Fprintf(os.Stderr, "cleanup container %s: %s (%v)\n", ids[i], out, err)
			}
		}
	}()
	start := func(image, port string, env ...string) (string, error) {
		argv := []string{"run", "--detach", "--publish", "127.0.0.1::" + port, "--label", "sqlc.test-local=true"}
		for _, v := range env {
			argv = append(argv, "--env", v)
		}
		argv = append(argv, image)
		out, err := exec.CommandContext(ctx, "docker", argv...).Output()
		if err != nil {
			return "", commandError(err)
		}
		id := strings.TrimSpace(string(out))
		if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(id) {
			return "", fmt.Errorf("invalid container id: %q", id)
		}
		ids = append(ids, id)
		out, err = exec.CommandContext(ctx, "docker", "port", id, port+"/tcp").Output()
		if err != nil {
			return "", commandError(err)
		}
		address := strings.TrimSpace(string(out))
		host, _, err := net.SplitHostPort(address)
		if err != nil || host != "127.0.0.1" {
			return "", fmt.Errorf("unexpected container address %q", address)
		}
		return address, nil
	}
	pgAddr, err := start("postgres:16", "5432", "POSTGRES_PASSWORD=sqlc-local-test")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	myAddr, err := start("mysql:9", "3306", "MYSQL_ROOT_PASSWORD=sqlc-local-test", "MYSQL_DATABASE=dinotest")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	pgURI := "postgres://postgres:sqlc-local-test@" + pgAddr + "/postgres?sslmode=disable"
	myURI := "root:sqlc-local-test@tcp(" + myAddr + ")/dinotest?multiStatements=true&parseTime=true"
	ready, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := waitReady(ready, func(ctx context.Context) error {
		conn, err := pgx.Connect(ctx, pgURI)
		if err != nil {
			return err
		}
		defer conn.Close(ctx)
		return conn.Ping(ctx)
	}); err != nil {
		fmt.Fprintln(os.Stderr, "PostgreSQL:", err)
		return 1
	}
	db, err := sql.Open("mysql", myURI)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer db.Close()
	if err := waitReady(ready, db.PingContext); err != nil {
		fmt.Fprintln(os.Stderr, "MySQL:", err)
		return 1
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "POSTGRESQL_SERVER_URI=") && !strings.HasPrefix(entry, "MYSQL_SERVER_URI=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "POSTGRESQL_SERVER_URI="+pgURI, "MYSQL_SERVER_URI="+myURI)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() > 0 {
			return exit.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func waitReady(ctx context.Context, check func(context.Context) error) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		attempt, cancel := context.WithTimeout(ctx, time.Second)
		err := check(attempt)
		cancel()
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("database not ready: %w (last error: %v)", ctx.Err(), err)
		case <-ticker.C:
		}
	}
}

func commandError(err error) error {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return fmt.Errorf("docker: %w: %s", err, exit.Stderr)
	}
	return err
}
