package scanner

import "errors"

var (
	ErrInvalidRequest = errors.New("invalid scan request")
	ErrEmptyTargets   = errors.New("empty targets")
	ErrScopeDenied    = errors.New("scope denied")
)
