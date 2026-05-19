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
	"github.com/P0m32Kun/Anchor/internal/builtin"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/worker"
)

func main() {
	workerMode := flag.Bool("worker", false, "run in worker mode")
	coreURL := flag.String("core-url", "", "core server URL (worker mode)")
	flag.Parse()

	// 如果 flag 未设置，从环境变量读取（Docker 场景下环境变量优先）
	if *coreURL == "" {
		*coreURL = os.Getenv("ANCHOR_CORE_URL")
	}

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

	// Sync vulnerability template knowledge base from the repo seed JSON.
	// Non-fatal: any failure is logged and execution continues with whatever
	// already exists in the database.
	seedPath := os.Getenv("ANCHOR_TEMPLATES_SEED")
	if seedPath == "" {
		seedPath = "docs/templates/vuln-templates.json"
	}
	if res, err := queries.SyncFindingTemplatesFromFile(seedPath); err != nil {
		log.Printf("[seed] finding templates sync (%s): %v", seedPath, err)
	} else if res != nil && (res.Inserted+res.Updated+res.Preserved+res.Deleted+res.Skipped) > 0 {
		log.Printf("[seed] finding templates: +%d ~%d ✓%d -%d ⚙%d (insert/update/preserve/delete/skip)",
			res.Inserted, res.Updated, res.Preserved, res.Deleted, res.Skipped)
	}

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
	if err := builtin.SyncAll(); err != nil {
		log.Printf("[worker] builtin sync: %v", err)
	}

	apiToken := os.Getenv("ANCHOR_API_TOKEN")
	ws := worker.NewWorkerServer(dataDir, coreURL, apiToken)

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
		if apiToken == "" {
			log.Fatal("[worker] ANCHOR_API_TOKEN environment variable is required for remote mode")
		}
		remoteClient = worker.NewRemoteClient(coreURL, endpoint, apiToken, dataDir)

		// 指数退避重试注册（最多 5 次，Server 可能还没启动）
		var regErr error
		for attempt := 1; attempt <= 5; attempt++ {
			regErr = remoteClient.Register("remote-worker")
			if regErr == nil {
				break
			}
			log.Printf("[worker] registration attempt %d/5 failed: %v", attempt, regErr)
			if attempt < 5 {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
			}
		}

		if regErr != nil {
			log.Printf("[worker] registration failed after 5 attempts: %v", regErr)
			log.Printf("[worker] continuing in standalone mode")
		} else {
			remoteClient.StartHeartbeat(30 * time.Second)
			remoteClient.StartBundleSync(60 * time.Second)
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
