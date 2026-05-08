package db

import (
	"database/sql"
)

type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type Queries struct{ db DBTX }

func New(db DBTX) *Queries { return &Queries{db: db} }

func nullableString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func sqlNullString(s *string) sql.NullString {
	if s != nil && *s != "" {
		return sql.NullString{String: *s, Valid: true}
	}
	return sql.NullString{}
}

func sqlNullStringValue(s string) sql.NullString {
	if s != "" {
		return sql.NullString{String: s, Valid: true}
	}
	return sql.NullString{}
}

func nullableBool(nb sql.NullBool) *bool {
	if nb.Valid {
		v := nb.Bool
		return &v
	}
	return nil
}

// WithTx runs fn inside a transaction. rawDB must be *sql.DB.
func WithTx(rawDB *sql.DB, fn func(*Queries) error) error {
	tx, err := rawDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(New(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
