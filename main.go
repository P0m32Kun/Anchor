package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"secbench/internal/api"
	"secbench/internal/db"
)

func main() {
	dataDir := os.Getenv("SECBENCH_DATA_DIR")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("cannot determine data dir:", err)
		}
		dataDir = filepath.Join(home, ".secbench")
	}

	sqliteDB, err := db.Open(dataDir)
	if err != nil {
		log.Fatal("open db:", err)
	}
	defer sqliteDB.Close()

	queries := db.New(sqliteDB)
	server := api.NewServer(queries, dataDir)

	mux := http.NewServeMux()
	server.Register(mux)

	port := os.Getenv("SECBENCH_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("secbench server listening on :%s", port)
	log.Printf("data dir: %s", dataDir)

	handler := api.CORSMiddleware(mux)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("server:", err)
	}
}
