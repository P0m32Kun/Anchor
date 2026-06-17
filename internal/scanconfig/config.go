package scanconfig

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/P0m32Kun/Anchor/internal/models"
	"gopkg.in/yaml.v3"
)

// Config holds deployment-level scan defaults loaded from scan.config.yaml and sidecar files.
type Config struct {
	Paths   PathsConfig   `yaml:"paths"`
	Passive PassiveConfig `yaml:"passive"`
	// NucleiTechRouting maps tech names to nuclei tag buckets (see nuclei_routing.go).
	NucleiTechRouting map[string]interface{} `yaml:"nuclei_tech_routing"`
	// PresetOverrides holds YAML preset overlays (merged at Preset() time).
	PresetOverrides map[string]interface{} `yaml:"presets"`

	// Resolved at load time
	HighRiskPorts       string
	ExcludeDomains      []string
	JunkKeywords        []JunkKeyword
	FfufDictionaryDefault string
}

type PathsConfig struct {
	HighRiskPortsFile     string `yaml:"high_risk_ports_file"`
	ExcludeDomainsFile    string `yaml:"exclude_domains_file"`
	JunkKeywordsFile      string `yaml:"junk_keywords_file"`
	FfufDictionaryDefault string `yaml:"ffuf_dictionary_default"`
}

type PassiveConfig struct {
	ResultLimit  int      `yaml:"result_limit"`
	Concurrency  int      `yaml:"concurrency"`
	FofaQueries  []string `yaml:"fofa_queries"`
	HunterQueries []string `yaml:"hunter_queries"`
	QuakeQuery   string   `yaml:"quake_query"`
}

// JunkKeyword is a passive-search junk filter rule.
type JunkKeyword struct {
	Keyword      string
	WordBoundary bool
}

// DefaultsResponse is exposed via GET /scan/defaults for the frontend.
type DefaultsResponse struct {
	HighRiskPorts         string                         `json:"high_risk_ports"`
	FfufDictionaryDefault string                         `json:"ffuf_dictionary_default"`
	Presets               map[string]models.PipelineConfig `json:"presets"`
	JunkKeywordCount      int                            `json:"junk_keyword_count"`
	ExcludeDomainCount    int                            `json:"exclude_domain_count"`
	ConfigPath            string                         `json:"config_path,omitempty"`
}

var (
	mu       sync.RWMutex
	loaded   *Config
	loadPath string
)

// Load reads scan.config.yaml from dataDir (if present) and resolves sidecar files.
// Missing files fall back to compiled defaults in sibling packages.
func Load(dataDir string) (*Config, error) {
	cfg := defaultYAMLConfig()
	configPath := filepath.Join(dataDir, "scan.config.yaml")

	if b, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, fmt.Errorf("parse %s: %w", configPath, err)
		}
		loadPath = configPath
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", configPath, err)
	} else {
		loadPath = ""
		log.Printf("[scanconfig] no %s, using compiled defaults", configPath)
	}

	if err := cfg.resolveFiles(dataDir); err != nil {
		return nil, err
	}

	mu.Lock()
	loaded = cfg
	mu.Unlock()

	log.Printf("[scanconfig] loaded high_risk_ports=%d chars exclude_domains=%d junk_keywords=%d presets=%d",
		len(cfg.HighRiskPorts), len(cfg.ExcludeDomains), len(cfg.JunkKeywords), len(cfg.PresetOverrides))
	return cfg, nil
}

// Get returns the loaded config or nil if Load was not called.
func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return loaded
}

// LoadPath returns the path to scan.config.yaml if it was loaded.
func LoadPath() string {
	mu.RLock()
	defer mu.RUnlock()
	return loadPath
}

// NucleiRouter returns the tech routing table for nuclei batch buckets.
func (c *Config) NucleiRouter() *NucleiRouter {
	if c == nil {
		return DefaultNucleiRouter()
	}
	return NucleiTechRoutingFromMap(c.NucleiTechRouting)
}

func (c *Config) resolveFiles(dataDir string) error {
	resolve := func(name string) string {
		if name == "" {
			return ""
		}
		if filepath.IsAbs(name) {
			return name
		}
		return filepath.Join(dataDir, name)
	}

	if c.Paths.HighRiskPortsFile != "" {
		if ports, err := readPortsFile(resolve(c.Paths.HighRiskPortsFile)); err == nil && ports != "" {
			c.HighRiskPorts = ports
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("high_risk_ports_file: %w", err)
		}
	}
	if c.HighRiskPorts == "" {
		c.HighRiskPorts = compiledHighRiskPorts()
	}

	if c.Paths.ExcludeDomainsFile != "" {
		if domains, err := readLinesFile(resolve(c.Paths.ExcludeDomainsFile)); err == nil && len(domains) > 0 {
			c.ExcludeDomains = domains
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("exclude_domains_file: %w", err)
		}
	}
	if len(c.ExcludeDomains) == 0 {
		c.ExcludeDomains = compiledExcludeDomains()
	}

	if c.Paths.JunkKeywordsFile != "" {
		if kws, err := readJunkKeywordsFile(resolve(c.Paths.JunkKeywordsFile)); err == nil && len(kws) > 0 {
			c.JunkKeywords = kws
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("junk_keywords_file: %w", err)
		}
	}
	if len(c.JunkKeywords) == 0 {
		c.JunkKeywords = compiledJunkKeywords()
	}

	if c.Paths.FfufDictionaryDefault != "" {
		c.FfufDictionaryDefault = c.Paths.FfufDictionaryDefault
	}
	if c.FfufDictionaryDefault == "" {
		c.FfufDictionaryDefault = models.DefaultFfufDictionaryID
	}

	if c.Passive.ResultLimit <= 0 {
		c.Passive.ResultLimit = 500
	}
	if c.Passive.Concurrency <= 0 {
		c.Passive.Concurrency = 3
	}
	if len(c.Passive.FofaQueries) == 0 {
		c.Passive.FofaQueries = defaultFofaQueries()
	}
	if len(c.Passive.HunterQueries) == 0 {
		c.Passive.HunterQueries = defaultHunterQueries()
	}
	if c.Passive.QuakeQuery == "" {
		c.Passive.QuakeQuery = defaultQuakeQuery()
	}

	return nil
}

// Preset returns a PipelineConfig for the named scan mode preset merged onto models defaults.
// Legacy mode value "src_low_noise" is normalized before lookup.
func (c *Config) Preset(name string) models.PipelineConfig {
	nMode, nNoise := models.NormalizeScanMode(name, "")
	base := models.DefaultPipelineConfig()
	switch nMode {
	case "external":
		if nNoise == "low" {
			base = models.DefaultExternalLowNoisePipelineConfig()
		} else {
			base = models.DefaultExternalStandardPipelineConfig()
		}
	case "internal":
		// base already internal default
	}
	if c == nil || len(c.PresetOverrides) == 0 {
		return base
	}
	rawVal, ok := c.PresetOverrides[name]
	if !ok {
		return base
	}
	raw, err := json.Marshal(rawVal)
	if err != nil {
		log.Printf("[scanconfig] preset %q marshal: %v, using compiled default", name, err)
		return base
	}
	merged, err := mergePresetJSON(base, raw)
	if err != nil {
		log.Printf("[scanconfig] preset %q merge: %v, using compiled default", name, err)
		return base
	}
	return merged
}

// DefaultsAPI builds the public defaults payload.
func (c *Config) DefaultsAPI() DefaultsResponse {
	if c == nil {
		c = &Config{}
		_ = c.resolveFiles("")
	}
	resp := DefaultsResponse{
		HighRiskPorts:         c.HighRiskPorts,
		FfufDictionaryDefault: c.FfufDictionaryDefault,
		Presets:               make(map[string]models.PipelineConfig),
		JunkKeywordCount:      len(c.JunkKeywords),
		ExcludeDomainCount:    len(c.ExcludeDomains),
		ConfigPath:            LoadPath(),
	}
	// Always include the canonical presets
	names := map[string]bool{"external": true, "external_low": true, "external_standard": true, "internal": true}
	// Add any extra presets from YAML
	for name := range c.PresetOverrides {
		names[name] = true
	}
	for name := range names {
		resp.Presets[name] = c.Preset(name)
	}
	return resp
}

func mergePresetJSON(base models.PipelineConfig, raw json.RawMessage) (models.PipelineConfig, error) {
	b, err := json.Marshal(base)
	if err != nil {
		return base, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return base, err
	}
	var overlay map[string]json.RawMessage
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return base, err
	}
	for k, v := range overlay {
		m[k] = v
	}
	merged, err := json.Marshal(m)
	if err != nil {
		return base, err
	}
	var out models.PipelineConfig
	if err := json.Unmarshal(merged, &out); err != nil {
		return base, err
	}
	return out, nil
}

func defaultYAMLConfig() *Config {
	return &Config{
		Paths: PathsConfig{
			HighRiskPortsFile:     "high-risk-ports.txt",
			ExcludeDomainsFile:    "exclude-domains.txt",
			JunkKeywordsFile:      "junk-keywords.txt",
			FfufDictionaryDefault: models.DefaultFfufDictionaryID,
		},
		Passive: PassiveConfig{
			ResultLimit:   500,
			Concurrency:   3,
			FofaQueries:   defaultFofaQueries(),
			HunterQueries: defaultHunterQueries(),
			QuakeQuery:    defaultQuakeQuery(),
		},
		PresetOverrides: map[string]interface{}{},
	}
}

func defaultFofaQueries() []string {
	return []string{`org="{{company}}"`, `cert="{{company}}"`, `title="{{company}}"`}
}

func defaultHunterQueries() []string {
	return []string{`icp.name="{{company}}"`, `cert="{{company}}"`}
}

func defaultQuakeQuery() string {
	return `cert:"{{company}}" OR title:"{{company}}"`
}

// EnsureConfigFiles copies bundled configs from srcDir into dataDir when missing.
func EnsureConfigFiles(srcDir, dataDir string) {
	if srcDir == "" || dataDir == "" {
		return
	}
	files := []string{"scan.config.yaml", "high-risk-ports.txt", "exclude-domains.txt", "junk-keywords.txt"}
	for _, f := range files {
		dst := filepath.Join(dataDir, f)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		src := filepath.Join(srcDir, f)
		b, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.WriteFile(dst, b, 0644); err != nil {
			log.Printf("[scanconfig] copy %s: %v", f, err)
			continue
		}
		log.Printf("[scanconfig] installed default %s", dst)
	}
}
