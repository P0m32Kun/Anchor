package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/P0m32Kun/Anchor/internal/api"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

func main() {
	workerMode := flag.Bool("worker", false, "run in worker mode")
	coreURL := flag.String("core-url", "", "core server URL (worker mode)")
	flag.Parse()

	dataDir := os.Getenv("ANCHOR_DATA_DIR")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("cannot determine data dir:", err)
		}
		dataDir = filepath.Join(home, ".anchor")
	}

	if *workerMode {
		runWorker(dataDir, *coreURL)
		return
	}

	runServer(dataDir)
}

func runServer(dataDir string) {
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

func runWorker(dataDir, coreURL string) {
	ws := worker.NewWorkerServer(dataDir)

	mux := http.NewServeMux()
	ws.Register(mux)

	// Listen on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("worker listen:", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	endpoint := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Signal ready to parent process
	fmt.Printf("WORKER_READY %s\n", endpoint)
	log.Printf("[worker] listening on %s", endpoint)
	if coreURL != "" {
		log.Printf("[worker] core URL: %s", coreURL)
	}

	if err := http.Serve(listener, mux); err != nil {
		log.Fatal("worker server:", err)
	}
}
