package main

import (
	"context"
	"database/sql"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: imgstore <command> [args...]")
	}

	db, err := sql.Open("sqlite3", "./store.db?_journal_mode=WAL")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := initSchema(db); err != nil {
		log.Fatal(err)
	}

	svc := NewService(db, "./store")
	if err := svc.Init(); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	
	switch os.Args[1] {
	case "fetch":
		if len(os.Args) != 5 {
			log.Fatal("Usage: imgstore fetch <name> <url> <checksum>")
		}
		name, url, checksum := os.Args[2], os.Args[3], os.Args[4]
		if err := svc.EnqueueImage(ctx, name, url, checksum); err != nil {
			log.Fatal(err)
		}
		log.Printf("Enqueued image %s", name)
		
	case "status":
		if len(os.Args) != 3 {
			log.Fatal("Usage: imgstore status <name>")
		}
		name := os.Args[2]
		state, err := svc.GetImageStatus(name)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Image %s: %s", name, state)
		
	case "worker":
		log.Println("Starting worker...")
		svc.RunWorker(ctx)
		
	default:
		log.Fatal("Unknown command:", os.Args[1])
	}
}

func initSchema(db *sql.DB) error {
	schema, err := ioutil.ReadFile(filepath.Join("migrations", "001_init.sql"))
	if err != nil {
		return err
	}
	_, err = db.Exec(string(schema))
	return err
}