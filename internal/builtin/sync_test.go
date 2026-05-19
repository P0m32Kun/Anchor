package builtin

import (
	"testing"
)

func clearBuiltinEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"ANCHOR_BUILTIN_SYNC",
		"ANCHOR_BUILTIN_DICT_REPO",
		"ANCHOR_BUILTIN_TEMPLATES_REPO",
		"ANCHOR_BUILTIN_FINGER_REPO",
		"ANCHOR_BUILTIN_DICT_REF",
		"ANCHOR_BUILTIN_TEMPLATES_REF",
		"ANCHOR_BUILTIN_FINGER_REF",
		"ANCHOR_BUILTIN_DICT_ROOT",
		"ANCHOR_BUILTIN_TEMPLATES_ROOT",
		"ANCHOR_BUILTIN_FINGER_ROOT",
	} {
		t.Setenv(k, "")
	}
}

func TestSyncModeOffSkips(t *testing.T) {
	clearBuiltinEnv(t)
	t.Setenv("ANCHOR_BUILTIN_SYNC", "off")

	cfg := LoadConfig()
	if cfg.ShouldSync() {
		t.Fatal("expected no sync when off")
	}
	if err := SyncAll(); err != nil {
		t.Fatalf("SyncAll: %v", err)
	}
}

func TestShouldSyncOnStartAndAlways(t *testing.T) {
	for _, mode := range []string{"on-start", "always"} {
		t.Run(mode, func(t *testing.T) {
			clearBuiltinEnv(t)
			t.Setenv("ANCHOR_BUILTIN_SYNC", mode)
			if !LoadConfig().ShouldSync() {
				t.Fatalf("expected sync enabled for mode %q", mode)
			}
		})
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	clearBuiltinEnv(t)

	cfg := LoadConfig()

	if cfg.Mode != "on-start" {
		t.Errorf("Mode: got %q, want on-start", cfg.Mode)
	}
	if cfg.DictRepo != "https://github.com/RBKD-SEC/dict.git" {
		t.Errorf("DictRepo: got %q", cfg.DictRepo)
	}
	if cfg.TemplatesRepo != "https://github.com/RBKD-SEC/RBKD-templates.git" {
		t.Errorf("TemplatesRepo: got %q", cfg.TemplatesRepo)
	}
	if cfg.FingerRepo != "https://github.com/RBKD-SEC/finger.git" {
		t.Errorf("FingerRepo: got %q", cfg.FingerRepo)
	}
	if cfg.DictRef != "main" || cfg.TemplatesRef != "main" || cfg.FingerRef != "main" {
		t.Errorf("refs: dict=%q templates=%q finger=%q", cfg.DictRef, cfg.TemplatesRef, cfg.FingerRef)
	}
	if cfg.DictRoot != "/opt/dict" {
		t.Errorf("DictRoot: got %q", cfg.DictRoot)
	}
	if cfg.TemplatesRoot != "/opt/rbkd-templates" {
		t.Errorf("TemplatesRoot: got %q", cfg.TemplatesRoot)
	}
	if cfg.FingerRoot != "/opt/finger" {
		t.Errorf("FingerRoot: got %q", cfg.FingerRoot)
	}
	if !cfg.ShouldSync() {
		t.Fatal("default mode should enable sync")
	}
}

func TestLoadConfigEnvOverrides(t *testing.T) {
	clearBuiltinEnv(t)
	t.Setenv("ANCHOR_BUILTIN_SYNC", "always")
	t.Setenv("ANCHOR_BUILTIN_DICT_ROOT", "/tmp/custom-dict")

	cfg := LoadConfig()
	if cfg.Mode != "always" {
		t.Errorf("Mode: got %q", cfg.Mode)
	}
	if cfg.DictRoot != "/tmp/custom-dict" {
		t.Errorf("DictRoot: got %q", cfg.DictRoot)
	}
}

func TestHeadShortMissingRepo(t *testing.T) {
	if got := HeadShort(t.TempDir()); got != "" {
		t.Fatalf("HeadShort on non-repo: got %q, want empty", got)
	}
}
