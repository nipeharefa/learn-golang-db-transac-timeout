package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type MyLogger struct{}

func (m *MyLogger) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	fmt.Fprintf(os.Stdout, "%s\n", msg)
}

func connecToPostgreSQL() (*pgxpool.Pool, error) {
	cfg, _ := pgxpool.ParseConfig(os.Getenv("POSTGRES_DATABASE_URL"))
	cfg.ConnConfig.Logger = &MyLogger{}
	cfg.ConnConfig.LogLevel = pgx.LogLevelTrace
	conn, err := pgxpool.ConnectConfig(context.Background(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	return conn, nil
}

func tx(errChan chan error, ctx context.Context, conn *pgxpool.Pool) {
	err := conn.BeginFunc(ctx, func(t pgx.Tx) error {
		fmt.Println("prepare for long running")
		time.Sleep(10 * time.Second)

		row := t.QueryRow(ctx, "SELECT 1")

		var i int

		ee := row.Scan(&i)
		fmt.Println("completed", ee)
		return nil
	})

	errChan <- err
}

func gerak(ctx context.Context, errChan chan error) {
	fmt.Println("goroutine")
	time.Sleep(10 * time.Second)
	errChan <- nil
}

func main() {

	conn, err := connecToPostgreSQL()
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go tx(errChan, ctx, conn)

	var done bool

	// pooling sampai semua sampai waktu max
	// time.after
	for {
		select {
		case <-time.After(1 * time.Second):
			cancel()
			done = true
			// errChan <- errors.New("timeout")
		case <-ctx.Done():
			fmt.Println(<-errChan, "S->", ctx.Err())
			done = true
		case e := <-errChan:
			fmt.Println(e)
			done = true
		default:
			fmt.Println("still waiting")
		}

		if done {
			break
		}
	}
	close(errChan)
	fmt.Println("finished")
}
