package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"imgstore/internal/api"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var (
		dbPath   = flag.String("db", "./store.db", "SQLite database path")
		storePath = flag.String("store", "./store", "Storage root path")
		addr     = flag.String("addr", ":8080", "HTTP server address")
	)
	flag.Parse()

	// Initialize database
	db, err := sql.Open("sqlite3", *dbPath+"?_journal_mode=WAL")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := initSchema(db); err != nil {
		log.Fatal(err)
	}

	// Initialize service
	svc := NewService(db, *storePath)
	if err := svc.Init(); err != nil {
		log.Fatal(err)
	}

	// Start background worker
	ctx, cancel := context.WithCancel(context.Background())
	go svc.RunWorker(ctx)

	// Start API server
	server := api.NewServer(db, svc, *addr)
	
	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	<-sigCh
	log.Println("Shutting down...")
	
	cancel() // Stop worker
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	
	if err := server.Stop(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}