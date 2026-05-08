package models

import (
	"encoding/json"
)

// ToJSON marshals any value to json.RawMessage, ignoring errors.
func ToJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
