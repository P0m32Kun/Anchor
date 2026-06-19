package db

import (
	"database/sql"
	"testing"
)

// --- sqlNullIntValue ---

func TestSqlNullIntValue(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		wantNil bool
		wantVal int64
	}{
		{"zero returns null", 0, true, 0},
		{"positive", 42, false, 42},
		{"negative", -1, false, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sqlNullIntValue(tt.input)
			if got.Valid == tt.wantNil {
				t.Errorf("sqlNullIntValue(%d).Valid = %v, want %v", tt.input, got.Valid, !tt.wantNil)
			}
			if !tt.wantNil && got.Int64 != tt.wantVal {
				t.Errorf("sqlNullIntValue(%d).Int64 = %d, want %d", tt.input, got.Int64, tt.wantVal)
			}
		})
	}
}

// --- nullableBool ---

func TestNullableBool(t *testing.T) {
	tests := []struct {
		name    string
		input   sql.NullBool
		wantNil bool
		wantVal bool
	}{
		{"not valid", sql.NullBool{Bool: true, Valid: false}, true, false},
		{"true", sql.NullBool{Bool: true, Valid: true}, false, true},
		{"false", sql.NullBool{Bool: false, Valid: true}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nullableBool(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", *got)
				}
			} else {
				if got == nil {
					t.Fatal("expected non-nil, got nil")
				}
				if *got != tt.wantVal {
					t.Errorf("got %v, want %v", *got, tt.wantVal)
				}
			}
		})
	}
}

// --- WithTx ---

func TestWithTx_Commit(t *testing.T) {
	rawDB := openTestDB(t)

	err := WithTx(rawDB, func(q *Queries) error {
		_, err := q.db.Exec(`INSERT INTO projects (id, name, rate_limit, default_profile, created_at, updated_at) VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
			"proj-tx", "tx-test", 10, "standard")
		return err
	})
	if err != nil {
		t.Fatalf("WithTx: %v", err)
	}

	// Verify the row was committed
	var count int
	if err := rawDB.QueryRow(`SELECT COUNT(*) FROM projects WHERE id = ?`, "proj-tx").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestWithTx_Rollback(t *testing.T) {
	rawDB := openTestDB(t)

	err := WithTx(rawDB, func(q *Queries) error {
		_, err := q.db.Exec(`INSERT INTO projects (id, name, rate_limit, default_profile, created_at, updated_at) VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
			"proj-rb", "rb-test", 10, "standard")
		return err
	})
	if err != nil {
		t.Fatalf("first WithTx: %v", err)
	}

	// Now try a tx that fails
	err = WithTx(rawDB, func(q *Queries) error {
		_, err := q.db.Exec(`INSERT INTO projects (id, name, rate_limit, default_profile, created_at, updated_at) VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
			"proj-rb-fail", "rb-fail", 10, "standard")
		if err != nil {
			return err
		}
		return sql.ErrConnDone // force rollback
	})
	if err == nil {
		t.Fatal("expected error from failing tx")
	}

	// Verify the failing tx row was rolled back
	var count int
	if err := rawDB.QueryRow(`SELECT COUNT(*) FROM projects WHERE id = ?`, "proj-rb-fail").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 (should be rolled back)", count)
	}
}
