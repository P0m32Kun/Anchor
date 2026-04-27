package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/P0m32Kun/Anchor/internal/api"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

func main() {
	workerMode := flag.Bool("worker", false, "run in worker mode")
	coreURL := flag.String("core-url", "", "core server URL (worker mode)")
	noLocalWorker := flag.Bool("no-local-worker", false, "do not auto-start local worker (server mode)")
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

	runServer(dataDir, !*noLocalWorker)
}

func runServer(dataDir string, autoStartWorker bool) {
	sqliteDB, err := db.Open(dataDir)
	if err != nil {
		log.Fatal("open db:", err)
	}
	defer sqliteDB.Close()

	queries := db.New(sqliteDB)
	server := api.NewServer(queries, sqliteDB, dataDir, autoStartWorker)

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

	// Listen on all interfaces so Docker/container can access
	listenAddr := "0.0.0.0:0"
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal("worker listen:", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	// Determine endpoint advertised to core server
	host := "127.0.0.1"
	if coreURL != "" {
		// In remote mode, advertise the container's accessible address
		// Core server will use this to fetch files/screenshots from worker
		host = os.Getenv("ANCHOR_WORKER_HOST")
		if host == "" {
			host = "127.0.0.1"
		}
	}
	endpoint := fmt.Sprintf("http://%s:%d", host, port)

	// Signal ready to parent process
	fmt.Printf("WORKER_READY %s\n", endpoint)
	log.Printf("[worker] listening on %s", endpoint)

	// Start remote client if core URL is provided
	var remoteClient *worker.RemoteClient
	if coreURL != "" {
		log.Printf("[worker] connecting to core: %s", coreURL)
		remoteClient = worker.NewRemoteClient(coreURL, endpoint)
		if err := remoteClient.Register("remote-worker"); err != nil {
			log.Printf("[worker] registration failed: %v", err)
			log.Printf("[worker] continuing in standalone mode")
		} else {
			remoteClient.StartHeartbeat(30 * time.Second)
			remoteClient.StartPolling()
			log.Printf("[worker] remote mode active")
		}
	}

	if err := http.Serve(listener, mux); err != nil {
		log.Fatal("worker server:", err)
	}

	if remoteClient != nil {
		remoteClient.Stop()
	}
}
