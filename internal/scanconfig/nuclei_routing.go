package scanconfig

import (
	"strings"
)

// NucleiRouteEntry describes one tech→template-set mapping.
type NucleiRouteEntry struct {
	Tags         []string `yaml:"tags"`
	MaxTemplates int      `yaml:"max_templates"`
	Action       string   `yaml:"action"` // "skip" for _default
}

// NucleiRouter resolves httpx technologies to nuclei template buckets.
type NucleiRouter struct {
	routes   map[string]NucleiRouteEntry
	defaultR NucleiRouteEntry
	fallback NucleiRouteEntry
}

// NucleiTechRoutingFromMap builds a router from scan.config.yaml nuclei_tech_routing.
func NucleiTechRoutingFromMap(raw map[string]interface{}) *NucleiRouter {
	if len(raw) == 0 {
		return DefaultNucleiRouter()
	}
	r := &NucleiRouter{routes: make(map[string]NucleiRouteEntry)}
	for name, v := range raw {
		entry := parseRouteEntry(v)
		switch name {
		case "_default":
			r.defaultR = entry
		case "_fallback":
			r.fallback = entry
		default:
			r.routes[strings.ToLower(name)] = entry
		}
	}
	if len(r.routes) == 0 && r.defaultR.Action == "" && len(r.fallback.Tags) == 0 {
		return DefaultNucleiRouter()
	}
	return r
}

func parseRouteEntry(v interface{}) NucleiRouteEntry {
	m, ok := v.(map[string]interface{})
	if !ok {
		return NucleiRouteEntry{}
	}
	entry := NucleiRouteEntry{}
	if tags, ok := m["tags"].([]interface{}); ok {
		for _, t := range tags {
			if s, ok := t.(string); ok && s != "" {
				entry.Tags = append(entry.Tags, s)
			}
		}
	}
	if mt, ok := m["max_templates"].(int); ok {
		entry.MaxTemplates = mt
	}
	if action, ok := m["action"].(string); ok {
		entry.Action = action
	}
	return entry
}

// DefaultNucleiRouter returns compiled defaults when YAML section is absent.
func DefaultNucleiRouter() *NucleiRouter {
	return &NucleiRouter{
		routes: map[string]NucleiRouteEntry{
			"jenkins": {Tags: []string{"jenkins"}, MaxTemplates: 25},
			"nginx":   {Tags: []string{"nginx", "nginx-version"}, MaxTemplates: 10},
			"apache":  {Tags: []string{"apache", "apache-detect"}, MaxTemplates: 10},
			"tomcat":  {Tags: []string{"tomcat"}, MaxTemplates: 15},
		},
		defaultR: NucleiRouteEntry{Action: "skip"},
		fallback: NucleiRouteEntry{Tags: []string{"tech-detect"}, MaxTemplates: 5},
	}
}

// Resolve picks a nuclei bucket for the given technologies and noise level.
// Returns bucket key (without "nuclei:" prefix), tags, and skip=true when no scan should run.
func (r *NucleiRouter) Resolve(technologies []string, noiseLevel string) (bucket string, tags []string, skip bool) {
	if r == nil {
		r = DefaultNucleiRouter()
	}
	noiseLevel = strings.ToLower(strings.TrimSpace(noiseLevel))
	if noiseLevel == "" {
		noiseLevel = "low"
	}

	if len(technologies) == 0 {
		if noiseLevel == "low" || r.defaultR.Action == "skip" {
			return "", nil, true
		}
		if len(r.defaultR.Tags) > 0 {
			return "_default", append([]string(nil), r.defaultR.Tags...), false
		}
		return "", nil, true
	}

	for _, tech := range technologies {
		key := normalizeTech(tech)
		if routeKey, entry, ok := r.matchRoute(key); ok {
			if entry.Action == "skip" {
				continue
			}
			return routeKey, append([]string(nil), entry.Tags...), false
		}
	}

	if noiseLevel == "low" {
		if len(r.fallback.Tags) > 0 {
			return "_fallback", append([]string(nil), r.fallback.Tags...), false
		}
		return "", nil, true
	}
	if len(r.fallback.Tags) > 0 {
		return "_fallback", append([]string(nil), r.fallback.Tags...), false
	}
	return "", nil, true
}

func (r *NucleiRouter) matchRoute(tech string) (routeKey string, entry NucleiRouteEntry, ok bool) {
	if entry, ok := r.routes[tech]; ok {
		return tech, entry, true
	}
	for key, entry := range r.routes {
		if strings.Contains(tech, key) {
			return key, entry, true
		}
	}
	return "", NucleiRouteEntry{}, false
}

// TagsForBucket returns nuclei -tags for a pooled batch bucket key (e.g. "nuclei:jenkins").
func (r *NucleiRouter) TagsForBucket(bucketKey string) []string {
	if r == nil {
		r = DefaultNucleiRouter()
	}
	name := strings.TrimPrefix(bucketKey, "nuclei:")
	switch name {
	case "_default":
		return append([]string(nil), r.defaultR.Tags...)
	case "_fallback":
		return append([]string(nil), r.fallback.Tags...)
	default:
		if entry, ok := r.routes[name]; ok {
			return append([]string(nil), entry.Tags...)
		}
	}
	return nil
}

func normalizeTech(tech string) string {
	return strings.ToLower(strings.TrimSpace(tech))
}
