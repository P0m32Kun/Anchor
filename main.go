package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/P0m32Kun/Anchor/internal/api"
	"github.com/P0m32Kun/Anchor/internal/db"
)

func main() {
	dataDir := os.Getenv("ANCHOR_DATA_DIR")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("cannot determine data dir:", err)
		}
		dataDir = filepath.Join(home, ".anchor")
	}

	sqliteDB, err := db.Open(dataDir)
	if err != nil {
		log.Fatal("open db:", err)
	}
	defer sqliteDB.Close()

	queries := db.New(sqliteDB)
	server := api.NewServer(queries, sqliteDB, dataDir)

	mux := http.NewServeMux()
	server.Register(mux)

	port := os.Getenv("ANCHOR_PORT")
	if port == "" {
		port = "17421"
	}

	log.Printf("anchor server listening on :%s", port)
	log.Printf("data dir: %s", dataDir)

	handler := api.CORSMiddleware(mux)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("server:", err)
	}
}
