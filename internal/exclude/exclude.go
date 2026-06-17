package exclude

import (
	"net/url"
	"strings"
	"sync"
)

// Manager manages the global domain exclusion list.
// It combines built-in defaults with user-configured custom domains.
type Manager struct {
	mu          sync.RWMutex
	custom      map[string]string // domain -> reason
	dirty       bool
	onChange    func()
}

// NewManager creates a new exclusion manager.
func NewManager() *Manager {
	return &Manager{
		custom: make(map[string]string),
	}
}

// SetOnChange sets a callback that is called when the exclusion list changes.
func (m *Manager) SetOnChange(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = fn
}

// IsExcluded checks if a domain should be excluded from scanning.
// It handles:
// - Exact domain matches
// - Subdomain matches (e.g., "api.github.com" matches "github.com")
// - URL inputs (extracts host from URL)
func (m *Manager) IsExcluded(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return false
	}

	domain := extractDomain(input)
	if domain == "" {
		return false
	}

	// Check built-in defaults first
	if isDefaultDomain(domain) {
		return true
	}

	// Check custom exclusions
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isCustomExcluded(domain)
}

// isDefaultDomain checks if domain matches the built-in exclusion list.
// Supports subdomain matching (e.g., "api.github.com" matches "github.com").
func isDefaultDomain(domain string) bool {
	// Direct match
	if IsDefaultDomain(domain) {
		return true
	}

	// Subdomain match: check if any parent domain is in the default list
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[i:], ".")
		if IsDefaultDomain(parent) {
			return true
		}
	}

	return false
}

// isCustomExcluded checks if domain matches any custom exclusion rule.
// Supports:
// - Exact match: "example.com"
// - Wildcard prefix: "*.example.com" matches "sub.example.com" only
// - Subdomain match: "example.com" matches "sub.example.com", "a.b.example.com"
func (m *Manager) isCustomExcluded(domain string) bool {
	for rule := range m.custom {
		if matchDomainRule(domain, rule) {
			return true
		}
	}
	return false
}

// matchDomainRule checks if domain matches a rule.
func matchDomainRule(domain, rule string) bool {
	rule = strings.ToLower(strings.TrimSpace(rule))

	// Exact match
	if domain == rule {
		return true
	}

	// Wildcard prefix: *.example.com
	if strings.HasPrefix(rule, "*.") {
		suffix := rule[2:]
		if strings.HasSuffix(domain, "."+suffix) {
			prefix := strings.TrimSuffix(domain, "."+suffix)
			if prefix != "" && !strings.Contains(prefix, ".") {
				return true
			}
		}
		return false
	}

	// Subdomain match: example.com matches sub.example.com, a.b.example.com
	if strings.HasSuffix(domain, "."+rule) {
		return true
	}

	return false
}

// extractDomain extracts the domain from input.
// Handles URLs, domain:port, and plain domains.
func extractDomain(input string) string {
	// Try parsing as URL
	if strings.Contains(input, "://") {
		u, err := url.Parse(input)
		if err == nil && u.Host != "" {
			host := u.Host
			// Strip port
			if idx := strings.LastIndex(host, ":"); idx > 0 {
				host = host[:idx]
			}
			return strings.ToLower(host)
		}
	}

	// Strip port if present (e.g., "example.com:8080")
	domain := input
	if idx := strings.LastIndex(domain, ":"); idx > 0 {
		// Make sure it's a port, not IPv6
		afterColon := domain[idx+1:]
		if !strings.Contains(afterColon, ":") && !strings.Contains(afterColon, "/") {
			domain = domain[:idx]
		}
	}

	return strings.ToLower(strings.TrimSpace(domain))
}

// AddCustom adds a domain to the custom exclusion list.
func (m *Manager) AddCustom(domain, reason string) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.custom[domain] = reason
	m.dirty = true
	if m.onChange != nil {
		m.onChange()
	}
}

// RemoveCustom removes a domain from the custom exclusion list.
func (m *Manager) RemoveCustom(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.custom[domain]; ok {
		delete(m.custom, domain)
		m.dirty = true
		if m.onChange != nil {
			m.onChange()
		}
		return true
	}
	return false
}

// SetCustomList replaces the entire custom exclusion list.
func (m *Manager) SetCustomList(domains map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.custom = domains
	m.dirty = true
	if m.onChange != nil {
		m.onChange()
	}
}

// GetCustomList returns a copy of the custom exclusion list.
func (m *Manager) GetCustomList() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]string, len(m.custom))
	for k, v := range m.custom {
		result[k] = v
	}
	return result
}

// GetAllExcludedDomains returns all excluded domains (defaults + custom).
func (m *Manager) GetAllExcludedDomains() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, 0, len(DefaultDomains)+len(m.custom))
	result = append(result, DefaultDomains...)
	for d := range m.custom {
		result = append(result, d)
	}
	return result
}

// IsDirty returns true if the custom list has been modified since last load/save.
func (m *Manager) IsDirty() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dirty
}

// ClearDirty clears the dirty flag.
func (m *Manager) ClearDirty() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dirty = false
}

// builtinDomainsVar is the mutable list of builtin excluded domains.
var builtinDomainsVar []string

// BuiltinDomainsFunc returns the current builtin excluded domains.
func BuiltinDomainsFunc() []string {
	return builtinDomainsVar
}

// SetBuiltinDomains overrides the builtin excluded domains at runtime.
func SetBuiltinDomains(domains []string) {
	builtinDomainsVar = domains
}
