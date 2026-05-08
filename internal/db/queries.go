package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type Queries struct{ db DBTX }

func New(db DBTX) *Queries { return &Queries{db: db} }
