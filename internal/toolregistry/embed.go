package toolregistry

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/P0m32Kun/Anchor/tools"
	"gopkg.in/yaml.v3"
)

// Decode decodes a single YAML byte slice into a ToolDef.
func Decode(data []byte) (*ToolDef, error) {
	var def ToolDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("decode tool yaml: %w", err)
	}
	return &def, nil
}

// DecodeAll walks the embedded tools/ FS and decodes all *.yaml files.
func DecodeAll() ([]*ToolDef, error) {
	entries, err := fs.ReadDir(tools.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("read tools dir: %w", err)
	}
	var defs []*ToolDef
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := tools.FS.ReadFile(e.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		def, err := Decode(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		defs = append(defs, def)
	}
	return defs, nil
}

// DefaultRegistry loads all embedded tools/*.yaml and returns a Registry.
// Panics on load failure — use at startup only.
func DefaultRegistry() *Registry {
	defs, err := DecodeAll()
	if err != nil {
		panic(fmt.Sprintf("toolregistry: default load failed: %v", err))
	}

	// ANCHOR_TOOLS_DIR: optional directory with *.yaml overrides.
	// Files in this directory with the same tool ID replace the embedded def.
	// Useful for development without rebuilding.
	if dir := os.Getenv("ANCHOR_TOOLS_DIR"); dir != "" {
		overrides, err := loadYAMLFromDir(dir)
		if err != nil {
			log.Printf("[toolregistry] ANCHOR_TOOLS_DIR=%s: %v", dir, err)
		} else {
			for _, ov := range overrides {
				replaced := false
				for i, d := range defs {
					if d.ID == ov.ID {
						defs[i] = ov
						replaced = true
						log.Printf("[toolregistry] ANCHOR_TOOLS_DIR: overrode tool %q from %s", ov.ID, dir)
						break
					}
				}
				if !replaced {
					defs = append(defs, ov)
					log.Printf("[toolregistry] ANCHOR_TOOLS_DIR: added tool %q from %s", ov.ID, dir)
				}
			}
		}
	}

	return MustLoad(defs)
}

// loadYAMLFromDir reads all *.yaml files from a directory.
func loadYAMLFromDir(dir string) ([]*ToolDef, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	var defs []*ToolDef
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		def, err := Decode(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		defs = append(defs, def)
	}
	return defs, nil
}


