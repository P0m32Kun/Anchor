package toolregistry

import "fmt"

// ParamType enumerates allowed parameter types.
type ParamType string

const (
	ParamString     ParamType = "string"
	ParamInt        ParamType = "int"
	ParamStringList ParamType = "string_list"
	ParamPath       ParamType = "path"
	ParamEnum       ParamType = "enum"
)

// OutputDef describes a tool's output format.
type OutputDef struct {
	Format       string `yaml:"format"` // jsonl | xml | greppable | text
	ArtifactType string `yaml:"artifact_type,omitempty"`
}

// ParamDef describes a single CLI parameter.
type ParamDef struct {
	Type     ParamType `yaml:"type"`
	Flag     string    `yaml:"flag,omitempty"`
	Required bool      `yaml:"required,omitempty"`
	Default  string    `yaml:"default,omitempty"`
	Values   []string  `yaml:"values,omitempty"`

	// ValueFlags maps enum values to sets of [flag, value] pairs.
	// E.g. nuclei profile "deep" → [["-severity","critical,high,medium,low,info"], ["-timeout","10"]]
	ValueFlags map[string][][]string `yaml:"value_flags,omitempty"`

	// Preset references a named preset block in ToolDef.Presets.
	Preset string `yaml:"preset,omitempty"`
}

// PresetDef is a named set of flag-value options.
type PresetDef struct {
	Entries map[string]interface{} `yaml:",inline"`
}

// ToolDef is the top-level tool definition matching tools/*.yaml.
type ToolDef struct {
	ID                string               `yaml:"id"`
	Binary            string               `yaml:"binary"`
	Description       string               `yaml:"description,omitempty"`
	Output            OutputDef            `yaml:"output"`
	TimeoutDefaultSec int                  `yaml:"timeout_default_sec,omitempty"`
	Parameters        map[string]ParamDef  `yaml:"parameters"`
	Presets           map[string]PresetDef `yaml:"presets,omitempty"`
	Literals          [][]string           `yaml:"literals,omitempty"`
	System            bool                 `yaml:"system,omitempty"`
}

// Registry holds a set of compiled ToolDefs loaded from YAML.
type Registry struct {
	tools map[string]*ToolDef
}

// Load creates a Registry from a slice of parsed ToolDefs.
func Load(defs []*ToolDef) (*Registry, error) {
	r := &Registry{tools: make(map[string]*ToolDef, len(defs))}
	for _, d := range defs {
		if d.ID == "" {
			return nil, fmt.Errorf("toolregistry: tool with empty id")
		}
		if d.Binary == "" {
			return nil, fmt.Errorf("tool %q: empty binary", d.ID)
		}
		if _, dup := r.tools[d.ID]; dup {
			return nil, fmt.Errorf("toolregistry: duplicate tool id %q", d.ID)
		}
		r.tools[d.ID] = d
	}
	return r, nil
}

// MustLoad calls Load and panics on error. Convenience for init / embed.
func MustLoad(defs []*ToolDef) *Registry {
	r, err := Load(defs)
	if err != nil {
		panic(err)
	}
	return r
}

// Get returns the tool definition by ID, or nil.
func (r *Registry) Get(id string) *ToolDef {
	return r.tools[id]
}

// List returns all tool IDs.
func (r *Registry) List() []string {
	ids := make([]string, 0, len(r.tools))
	for id := range r.tools {
		ids = append(ids, id)
	}
	return ids
}

// Binaries returns the set of binary names used by all non-system tools.
func (r *Registry) Binaries() []string {
	seen := make(map[string]bool)
	var bins []string
	for _, d := range r.tools {
		if d.System {
			continue
		}
		if !seen[d.Binary] {
			seen[d.Binary] = true
			bins = append(bins, d.Binary)
		}
	}
	return bins
}
