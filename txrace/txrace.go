package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <dbname>", os.Args[0])
	}
	dbName := os.Args[1]
	db, err := sql.Open("postgres", "postgres://localhost/"+dbName)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sql := `
DROP TABLE IF EXISTS txrace_test;
CREATE TABLE txrace_test (
  t timestamptz NOT NULL
);
`
	if _, err := db.Exec(sql); err != nil {
		log.Fatalf("Error creating test table: %s", err)
	}

	sql = `
INSERT INTO txrace_test (t)
VALUES ($1);
`
	insert, err := db.Prepare(sql)
	if err != nil {
		log.Fatal(err)
	}
	sql = `
SELECT t
FROM txrace_test
LIMIT 1;
`
	query, err := db.Prepare(sql)
	if err != nil {
		log.Fatal(err)
	}

	var n int64
	var errs int64
	for i := 0; i < 10; i++ {
		go func() {
			for i := 0; ; i++ {
				if err := runTx(db, insert, query); err != nil {
					atomic.AddInt64(&errs, 1)
				}
				atomic.AddInt64(&n, 1)
			}
		}()
	}
	for range time.Tick(time.Second) {
		fmt.Printf(
			"%d (%d errors)\n",
			atomic.SwapInt64(&n, 0),
			atomic.SwapInt64(&errs, 0),
		)
	}
}

func runTx(db *sql.DB, insert, query *sql.Stmt) error {
	ctx, cancel := context.WithCancel(context.Background())
	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		timer := time.NewTimer(time.Microsecond)
		defer timer.Stop()
		select {
		case <-quit:
		case <-timer.C:
		}
		cancel()
		close(done)
	}()
	defer func() {
		close(quit)
		<-done
	}()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txInsert := tx.Stmt(insert)
	txQuery := tx.Stmt(query)
	for i := 0; i < 5; i++ {
		var t time.Time
		row := txQuery.QueryRow()
		if err := row.Scan(&t); err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("error running query: %s", err)
		}
		_ = t

		if _, err := txInsert.Exec(time.Now()); err != nil {
			return fmt.Errorf("error running exec: %s", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing: %s", err)
	}
	return nil
}
