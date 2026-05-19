package httpxfp

import "errors"

var (
	// ErrBuiltinReadOnly is returned when a write is attempted on a builtin fingerprint.
	ErrBuiltinReadOnly = errors.New("builtin fingerprint is read-only")
	// ErrNotBuiltin is returned when an operation is only valid for builtin rows.
	ErrNotBuiltin = errors.New("fingerprint is not builtin")
)
