package util

import (
	"fmt"
	"sync/atomic"
	"time"
)

var idCounter atomic.Int64

func GenerateID() string {
	return fmt.Sprintf("id-%d-%d", time.Now().UnixNano(), idCounter.Add(1))
}
