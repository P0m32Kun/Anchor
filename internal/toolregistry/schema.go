package toolregistry

import (
	"fmt"
	"sort"
	"strings"
)

// RenderParams is a flat map of parameter key → value for Render.
type RenderParams map[string]interface{}

// HighRiskPorts is the curated list of high-risk ports, matching
// internal/worker/commands.go's HighRiskPorts constant. Defined here
// in Go instead of inline YAML (per design §6.3: avoid 2KB YAML string).


// highRiskPortsVar is the mutable version of HighRiskPorts for runtime override.
var highRiskPortsVar = HighRiskPorts

// HighRiskPortsFunc returns the current high-risk ports string.
func HighRiskPortsFunc() string {
	return highRiskPortsVar
}

// SetHighRiskPorts overrides the high-risk ports at runtime.
func SetHighRiskPorts(ports string) {
	if ports != "" {
		highRiskPortsVar = ports
	}
}
const HighRiskPorts = "21,22,23,25,53,80,81,88,110,135,139,143,389,443,445,465,587,636,873,993,995," +
	"1080,1099,1433,1521,1723,2049,2082,2375,2376,2480,3000,3128,3306,3389," +
	"4040,4194,4369,4444,4848,5000,5432,5601,5672,5900,5901,5984,6379,6443," +
	"7000,7001,7002,7077,7474,8000,8001,8008,8009,8020,8060,8080,8081,8086,8088,8090,8161,8200,8443,8500,8531,8888,8983," +
	"9000,9001,9042,9043,9060,9080,9090,9091,9092,9100,9200,9300,9418,9443,9981," +
	"10000,10022,10250,10255,11211,11434,13306,15672,15692,16379,18080,18081,18082,18091," +
	"27017,27018,27019,28017,50000,50070,50075,61613,61616"

// Render generates the argv slice for the given tool_id and params.
// Argv[0] is the binary name, suitable for exec.
func (r *Registry) Render(id string, params RenderParams) ([]string, error) {
	def := r.tools[id]
	if def == nil {
		return nil, fmt.Errorf("tool %q not found in registry", id)
	}

	var argv []string
	argv = append(argv, def.Binary)

	// 1. Literals (fixed flags)
	for _, lit := range def.Literals {
		argv = append(argv, lit...)
	}

	// 2. Parameter flags — track which keys were already consumed so post-processing
	//    doesn't duplicate them.
	consumed := make(map[string]bool)
	for key, param := range def.Parameters {
		tokens := r.renderParam(id, def, key, param, params)
		argv = append(argv, tokens...)
		if len(tokens) > 0 {
			consumed[key] = true
		}
	}

	// 3. Post-processing: mode/conditional flags that can't be expressed solely
	//    in YAML without making the schema overly complex.
	switch id {
	case "subfinder":
		if mode, ok := toString(params["mode"], true); ok && mode == "passive" {
			argv = append(argv, "-passive")
		}
	case "dnsx":
		// When no record_types provided, add default -a -aaaa -cname
		rt, ok := toStringList(params["record_types"], true)
		if !ok || len(rt) == 0 {
			argv = append(argv, "-a", "-aaaa", "-cname")
		} else {
			for _, r := range rt {
				argv = append(argv, "-"+strings.ToLower(r))
			}
		}
	case "nmap_service":
		if v, ok := toInt(params["host_timeout"], true); ok && v > 0 && !consumed["host_timeout"] {
			argv = append(argv, "--host-timeout", fmt.Sprintf("%ds", v))
		}
	case "nuclei":
		// template_path overrides scan_depth+workflow_dir.
		// Generic parameter rendering handles tags/workflow_dir/template_path.
		// Post-processing only adds -w/-tags if NOT already consumed by the generic renderer.
		tp, _ := toString(params["template_path"], true)
		if tp != "" && !consumed["template_path"] {
			argv = append(argv, "-w", tp)
		}
		if !consumed["template_path"] {
			sd, _ := toString(params["scan_depth"], true)
			if sd == "" {
				sd = "tags"
			}
			wd, _ := toString(params["workflow_dir"], true)
			tags, _ := toStringList(params["tags"], true)
			switch sd {
			case "workflow":
				if wd != "" && !consumed["workflow_dir"] {
					argv = append(argv, "-w", wd)
				}
			case "both":
				if wd != "" && !consumed["workflow_dir"] {
					argv = append(argv, "-w", wd)
				}
				if len(tags) > 0 && !consumed["tags"] {
					argv = append(argv, "-tags", strings.Join(tags, ","))
				}
			default: // "tags"
				if len(tags) > 0 && !consumed["tags"] {
					argv = append(argv, "-tags", strings.Join(tags, ","))
				}
			}
		}
	case "gau":
		// Positional: domain goes last, no flag
		if d, ok := toString(params["domain"], true); ok && d != "" {
			// gau only takes positional argument
		}
	}

	return argv, nil
}

// renderParam handles a single parameter and returns the flag-value tokens.
func (r *Registry) renderParam(id string, def *ToolDef, key string, param ParamDef, params RenderParams) []string {
	rawVal, has := params[key]

	switch param.Type {
	case ParamInt:
		v, ok := toInt(rawVal, has)
		if ok && v > 0 && param.Flag != "" {
			return []string{param.Flag, fmt.Sprintf("%d", v)}
		}
		return nil

	case ParamString, ParamPath:
		v, ok := toString(rawVal, has)
		if ok && v != "" {
			if param.Flag != "" {
				return []string{param.Flag, v}
			}
			// Positional (no flag): return just the value
			return []string{v}
		}
		return nil

	case ParamStringList:
		v, ok := toStringList(rawVal, has)
		if ok && len(v) > 0 && param.Flag != "" {
			return []string{param.Flag, strings.Join(v, ",")}
		}
		return nil

	case ParamEnum:
		v, ok := toString(rawVal, has)
		if !ok || v == "" {
			if param.Default != "" {
				v = param.Default
			} else {
				return nil
			}
		}

		// ValueFlags: e.g. nuclei "light" → [-severity critical,high -timeout 3]
		if vf, ok2 := param.ValueFlags[v]; ok2 {
			var tokens []string
			for _, pair := range vf {
				tokens = append(tokens, pair...)
			}
			return tokens
		}

		// Preset: e.g. naabu "port_range" → [-tp 1000] or [-p <ports>]
		if param.Preset != "" {
			return r.renderPreset(def, param.Preset, v)
		}

		// Simple enum with flag: not common, but handle it
		if param.Flag != "" {
			return []string{param.Flag, v}
		}
		return nil
	}

	return nil
}

// renderPreset expands a named preset for the given value.
func (r *Registry) renderPreset(def *ToolDef, presetName, value string) []string {
	pd, ok := def.Presets[presetName]
	if !ok {
		return nil
	}

	entry, ok := pd.Entries[value]
	if !ok {
		// Try "default" entry for custom values
		entry, ok = pd.Entries["default"]
		if !ok {
			return nil
		}
		// default is a map with "flag" key
		if m, ok2 := entry.(map[string]interface{}); ok2 {
			if flag, ok3 := m["flag"]; ok3 {
				return []string{fmt.Sprintf("%v", flag), value}
			}
		}
		return nil
	}

	// entry is a []interface{} of flag-value pairs
	switch items := entry.(type) {
	case []interface{}:
		var tokens []string
		for _, item := range items {
			switch pair := item.(type) {
			case []interface{}:
				for _, s := range pair {
					tokens = append(tokens, fmt.Sprintf("%v", s))
				}
			}
		}
		// Replace <HighRiskPorts> placeholder
		result := make([]string, len(tokens))
		for i, t := range tokens {
			if t == "<HighRiskPorts>" {
				result[i] = HighRiskPorts
			} else {
				result[i] = t
			}
		}
		return result
	}
	return nil
}

// --- set comparison utilities ---

// ArgvSet returns a sorted copy of argv for order-independent comparison.
func ArgvSet(argv []string) []string {
	out := make([]string, len(argv))
	copy(out, argv)
	sort.Strings(out)
	return out
}

// ArgvSetMinus returns a sorted copy with ignore tokens removed.
func ArgvSetMinus(argv, ignore []string) []string {
	ignoreSet := make(map[string]bool, len(ignore))
	for _, t := range ignore {
		ignoreSet[t] = true
	}
	var filtered []string
	for _, t := range argv {
		if !ignoreSet[t] {
			filtered = append(filtered, t)
		}
	}
	sort.Strings(filtered)
	return filtered
}

// --- type coercion helpers ---

func toInt(v interface{}, present bool) (int, bool) {
	if !present || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	}
	return 0, false
}

func toString(v interface{}, present bool) (string, bool) {
	if !present || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func toStringList(v interface{}, present bool) ([]string, bool) {
	if !present || v == nil {
		return nil, false
	}
	switch l := v.(type) {
	case []string:
		return l, len(l) > 0
	case []interface{}:
		var out []string
		for _, item := range l {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out, len(out) > 0
	}
	if s, ok := v.(string); ok && s != "" {
		return strings.Split(s, ","), true
	}
	return nil, false
}
