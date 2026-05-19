package builtin

import "os"

// Config holds builtin asset Git sync settings from environment variables.
type Config struct {
	Mode          string // off | on-start | always
	DictRepo      string
	TemplatesRepo string
	FingerRepo    string
	DictRef       string
	TemplatesRef  string
	FingerRef     string
	DictRoot      string
	TemplatesRoot string
	FingerRoot    string
}

// ShouldSync reports whether SyncAll should run Git operations.
// Only mode "off" disables sync.
func (c Config) ShouldSync() bool {
	return c.Mode != "off"
}

// LoadConfig reads builtin sync settings from the environment with spec defaults.
func LoadConfig() Config {
	return Config{
		Mode:          envOr("ANCHOR_BUILTIN_SYNC", "on-start"),
		DictRepo:      envOr("ANCHOR_BUILTIN_DICT_REPO", "https://github.com/RBKD-SEC/dict.git"),
		TemplatesRepo: envOr("ANCHOR_BUILTIN_TEMPLATES_REPO", "https://github.com/RBKD-SEC/RBKD-templates.git"),
		FingerRepo:    envOr("ANCHOR_BUILTIN_FINGER_REPO", "https://github.com/RBKD-SEC/finger.git"),
		DictRef:       envOr("ANCHOR_BUILTIN_DICT_REF", "main"),
		TemplatesRef:  envOr("ANCHOR_BUILTIN_TEMPLATES_REF", "main"),
		FingerRef:     envOr("ANCHOR_BUILTIN_FINGER_REF", "main"),
		DictRoot:      envOr("ANCHOR_BUILTIN_DICT_ROOT", "/opt/dict"),
		TemplatesRoot: envOr("ANCHOR_BUILTIN_TEMPLATES_ROOT", "/opt/rbkd-templates"),
		FingerRoot:    envOr("ANCHOR_BUILTIN_FINGER_ROOT", "/opt/finger"),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
