package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/klikz/api_v3/internal/routecomments"
	"github.com/klikz/api_v3/utils"
	_ "github.com/lib/pq"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "show updates without changing database")
	timeout := flag.Duration("timeout", 30*time.Second, "database operation timeout")
	flag.Parse()

	_ = godotenv.Load()

	connString, err := utils.DBConnString()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", connString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	count, err := routecomments.Apply(ctx, db, *dryRun)
	if err != nil {
		log.Fatal(err)
	}

	if *dryRun {
		fmt.Printf("DRY-RUN completed. %d route comments prepared.\n", count)
		return
	}
	fmt.Printf("Updated auth.routes comments: %d\n", count)
}
