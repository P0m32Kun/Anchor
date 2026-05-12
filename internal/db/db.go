package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func Open(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	dbPath := filepath.Join(dataDir, "anchor.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func Migrate(db *sql.DB) error { return migrate(db) }

func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	if version < 1 {
		if err := migrateV1(db); err != nil {
			return fmt.Errorf("migrate v1: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 1"); err != nil {
			return fmt.Errorf("set user_version 1: %w", err)
		}
		version = 1
	}

	if version < 2 {
		if err := migrateAddRateLimit(db); err != nil {
			return fmt.Errorf("migrate v2 (rate_limit): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 2"); err != nil {
			return fmt.Errorf("set user_version 2: %w", err)
		}
		version = 2
	}

	if version < 3 {
		if err := migrateV02(db); err != nil {
			return fmt.Errorf("migrate v3 (v0.2): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 3"); err != nil {
			return fmt.Errorf("set user_version 3: %w", err)
		}
		version = 3
	}

	if version < 4 {
		if err := migrateV04(db); err != nil {
			return fmt.Errorf("migrate v4 (v0.4 pipeline): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 4"); err != nil {
			return fmt.Errorf("set user_version 4: %w", err)
		}
		version = 4
	}

	if version < 5 {
		if err := migrateV05(db); err != nil {
			return fmt.Errorf("migrate v5 (drop start_time/end_time): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 5"); err != nil {
			return fmt.Errorf("set user_version 5: %w", err)
		}
		version = 5
	}

	if version < 6 {
		if err := migrateV06(db); err != nil {
			return fmt.Errorf("migrate v6 (pipeline_runs): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 6"); err != nil {
			return fmt.Errorf("set user_version 6: %w", err)
		}
		version = 6
	}

	if version < 7 {
		if err := migrateV07(db); err != nil {
			return fmt.Errorf("migrate v7 (pipeline_run_stages): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 7"); err != nil {
			return fmt.Errorf("set user_version 7: %w", err)
		}
		version = 7
	}

	if version < 8 {
		if err := migrateV08(db); err != nil {
			return fmt.Errorf("migrate v8 (pipeline_runs mode): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 8"); err != nil {
			return fmt.Errorf("set user_version 8: %w", err)
		}
		version = 8
	}

	if version < 9 {
		if err := migrateV09(db); err != nil {
			return fmt.Errorf("migrate v9 (engine_credentials): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 9"); err != nil {
			return fmt.Errorf("set user_version 9: %w", err)
		}
		version = 9
	}

	if version < 10 {
		if err := migrateV10(db); err != nil {
			return fmt.Errorf("migrate v10 (nuclei custom): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 10"); err != nil {
			return fmt.Errorf("set user_version 10: %w", err)
		}
		version = 10
	}

	if version < 11 {
		if err := migrateV11(db); err != nil {
			return fmt.Errorf("migrate v11 (drop fofa email): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 11"); err != nil {
			return fmt.Errorf("set user_version 11: %w", err)
		}
		version = 11
	}

	if version < 12 {
		if err := migrateV12(db); err != nil {
			return fmt.Errorf("migrate v12 (fix scan_tasks.run_id FK): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 12"); err != nil {
			return fmt.Errorf("set user_version 12: %w", err)
		}
		version = 12
	}

	if version < 13 {
		if err := migrateV13(db); err != nil {
			return fmt.Errorf("migrate v13 (scan_tasks.error_message): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 13"); err != nil {
			return fmt.Errorf("set user_version 13: %w", err)
		}
		version = 13
	}

	if version < 14 {
		if err := migrateV14(db); err != nil {
			return fmt.Errorf("migrate v14 (reports table): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 14"); err != nil {
			return fmt.Errorf("set user_version 14: %w", err)
		}
		version = 14
	}

	if version < 15 {
		if err := migrateV15(db); err != nil {
			return fmt.Errorf("migrate v15 (service_fingerprints product/version): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 15"); err != nil {
			return fmt.Errorf("set user_version 15: %w", err)
		}
		version = 15
	}

	if version < 16 {
		if err := migrateV16(db); err != nil {
			return fmt.Errorf("migrate v16 (dictionaries + httpx_fingerprints): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 16"); err != nil {
			return fmt.Errorf("set user_version 16: %w", err)
		}
		version = 16
	}

	if version < 17 {
		if err := migrateV17(db); err != nil {
			return fmt.Errorf("migrate v17 (slow_scan_tasks): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 17"); err != nil {
			return fmt.Errorf("set user_version 17: %w", err)
		}
		version = 17
	}

	if version < 18 {
		if err := migrateV18(db); err != nil {
			return fmt.Errorf("migrate v18 (finding_templates): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 18"); err != nil {
			return fmt.Errorf("set user_version 18: %w", err)
		}
		version = 18
	}

	if err := ensureProjectsColumns(db); err != nil {
		return fmt.Errorf("ensure projects columns: %w", err)
	}

	return nil
}
