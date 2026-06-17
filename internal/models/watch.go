package models

import "time"

// WatchProject extends Project with watch-mode fields for periodic scanning.
type WatchProject struct {
	ID           string     `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	WatchEnabled bool       `json:"watch_enabled" db:"watch_enabled"`
	WatchIntervalHours int  `json:"watch_interval_hours" db:"watch_interval_hours"`
	WatchPassiveOnly  bool  `json:"watch_passive_only" db:"watch_passive_only"`
	WatchLastTickAt   *time.Time `json:"watch_last_tick_at,omitempty" db:"watch_last_tick_at"`
}
