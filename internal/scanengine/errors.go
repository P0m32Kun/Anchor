package scanengine

import "errors"

// ErrAbsoluteTimeout is returned when a scan run hits the hard duration limit
// before all work items could be processed.
var ErrAbsoluteTimeout = errors.New("absolute timeout reached")
