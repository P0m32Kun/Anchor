package custom

import "errors"

var (
	// ErrBuiltinReadOnly is returned when a write is attempted on a builtin source.
	ErrBuiltinReadOnly = errors.New("builtin nuclei source is read-only")
	// ErrNotBuiltin is returned when an operation is only valid for builtin rows.
	ErrNotBuiltin = errors.New("nuclei source is not builtin")
)
