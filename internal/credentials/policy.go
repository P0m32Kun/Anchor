package credentials

import (
	"fmt"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/sources"
)

// PolicyConfig 策略配置
type PolicyConfig struct {
	// 平台特定配置
	PlatformConfigs map[string]PlatformPolicy `json:"platform_configs"`
	// 默认策略
	DefaultPolicy PlatformPolicy `json:"default_policy"`
}

// PlatformPolicy 平台策略
type PlatformPolicy struct {
	// 是否启用该平台
	Enabled bool `json:"enabled"`
	// 扫描速率限制（请求/秒）
	RateLimit int `json:"rate_limit"`
	// 并发数
	Concurrency int `json:"concurrency"`
	// 超时时间（秒）
	Timeout int `json:"timeout"`
	// 是否使用 session token
	UseSession bool `json:"use_session"`
	// 是否使用 API key
	UseAPIKey bool `json:"use_api_key"`
	// 自定义 headers
	Headers map[string]string `json:"headers,omitempty"`
}

// DefaultPolicyConfig 返回默认策略配置
func DefaultPolicyConfig() PolicyConfig {
	return PolicyConfig{
		PlatformConfigs: map[string]PlatformPolicy{
			"hackerone": {
				Enabled:     true,
				RateLimit:   10,
				Concurrency: 5,
				Timeout:     30,
				UseSession:  true,
				UseAPIKey:   true,
			},
			"bugcrowd": {
				Enabled:     true,
				RateLimit:   10,
				Concurrency: 5,
				Timeout:     30,
				UseSession:  true,
				UseAPIKey:   true,
			},
			"intigriti": {
				Enabled:     true,
				RateLimit:   5,
				Concurrency: 3,
				Timeout:     30,
				UseSession:  false,
				UseAPIKey:   true,
			},
			"yeswehack": {
				Enabled:     true,
				RateLimit:   5,
				Concurrency: 3,
				Timeout:     30,
				UseSession:  false,
				UseAPIKey:   true,
			},
		},
		DefaultPolicy: PlatformPolicy{
			Enabled:     true,
			RateLimit:   5,
			Concurrency: 3,
			Timeout:     30,
			UseSession:  true,
			UseAPIKey:   false,
		},
	}
}

// PolicyManager 策略管理器
type PolicyManager struct {
	config     PolicyConfig
	discoverer *Discoverer
	registry   *sources.Registry
}

// NewPolicyManager 创建策略管理器
func NewPolicyManager(config PolicyConfig) *PolicyManager {
	return &PolicyManager{
		config:     config,
		discoverer: NewDiscoverer(DefaultDiscoveryConfig()),
		registry:   sources.NewRegistry(),
	}
}

// GetPlatformPolicy 获取平台策略
func (pm *PolicyManager) GetPlatformPolicy(platform string) PlatformPolicy {
	platform = strings.ToLower(platform)

	// 查找平台特定配置
	if policy, ok := pm.config.PlatformConfigs[platform]; ok {
		return policy
	}

	// 返回默认策略
	return pm.config.DefaultPolicy
}

// GetPlatformPolicyWithCredential 获取平台策略（包含凭证信息）
func (pm *PolicyManager) GetPlatformPolicyWithCredential(platform string) (*PlatformPolicyWithCredential, error) {
	platform = strings.ToLower(platform)

	// 获取策略
	policy := pm.GetPlatformPolicy(platform)

	// 获取凭证
	cred, err := pm.discoverer.GetCredentialForPlatform(platform)
	if err != nil {
		// 没有凭证，返回策略但标记为无凭证
		return &PlatformPolicyWithCredential{
			Platform:       platform,
			Policy:         policy,
			HasCredential:  false,
			CredentialType: "none",
		}, nil
	}

	// 确定凭证类型
	credType := "none"
	if cred.Token != "" && policy.UseSession {
		credType = "session"
	} else if cred.APIKey != "" && policy.UseAPIKey {
		credType = "api_key"
	}

	return &PlatformPolicyWithCredential{
		Platform:       platform,
		Policy:         policy,
		HasCredential:  true,
		CredentialType: credType,
		Username:       cred.Username,
	}, nil
}

// PlatformPolicyWithCredential 带凭证信息的平台策略
type PlatformPolicyWithCredential struct {
	Platform       string        `json:"platform"`
	Policy         PlatformPolicy `json:"policy"`
	HasCredential  bool          `json:"has_credential"`
	CredentialType string        `json:"credential_type"` // "session", "api_key", "none"
	Username       string        `json:"username,omitempty"`
}

// ListEnabledPlatforms 列出所有启用的平台
func (pm *PolicyManager) ListEnabledPlatforms() []string {
	var platforms []string

	for platform, policy := range pm.config.PlatformConfigs {
		if policy.Enabled {
			platforms = append(platforms, platform)
		}
	}

	return platforms
}

// ListEnabledPlatformsWithCredentials 列出所有启用且有凭证的平台
func (pm *PolicyManager) ListEnabledPlatformsWithCredentials() []string {
	var platforms []string

	for platform, policy := range pm.config.PlatformConfigs {
		if !policy.Enabled {
			continue
		}

		// 检查是否有凭证
		if pm.discoverer.HasCredential(platform) {
			platforms = append(platforms, platform)
		}
	}

	return platforms
}

// GetScanConfigForPlatform 获取平台的扫描配置
func (pm *PolicyManager) GetScanConfigForPlatform(platform string) (*ScanConfig, error) {
	platform = strings.ToLower(platform)

	// 获取策略和凭证
	policyWithCred, err := pm.GetPlatformPolicyWithCredential(platform)
	if err != nil {
		return nil, fmt.Errorf("get policy: %w", err)
	}

	// 获取平台信息
	platformInfo, ok := pm.registry.Get(platform)
	if !ok {
		return nil, fmt.Errorf("platform not found: %s", platform)
	}

	// 构建扫描配置
	config := &ScanConfig{
		Platform:       platform,
		PlatformName:   platformInfo.Name,
		PlatformType:   string(platformInfo.Type),
		Domains:        platformInfo.Domains,
		RateLimit:      policyWithCred.Policy.RateLimit,
		Concurrency:    policyWithCred.Policy.Concurrency,
		Timeout:        policyWithCred.Policy.Timeout,
		HasCredential:  policyWithCred.HasCredential,
		CredentialType: policyWithCred.CredentialType,
		Username:       policyWithCred.Username,
	}

	return config, nil
}

// ScanConfig 扫描配置
type ScanConfig struct {
	Platform       string   `json:"platform"`
	PlatformName   string   `json:"platform_name"`
	PlatformType   string   `json:"platform_type"`
	Domains        []string `json:"domains"`
	RateLimit      int      `json:"rate_limit"`
	Concurrency    int      `json:"concurrency"`
	Timeout        int      `json:"timeout"`
	HasCredential  bool     `json:"has_credential"`
	CredentialType string   `json:"credential_type"`
	Username       string   `json:"username,omitempty"`
}

// ValidatePolicy 验证策略配置
func (pm *PolicyManager) ValidatePolicy() []ValidationError {
	var errors []ValidationError

	// 验证每个平台配置
	for platform, policy := range pm.config.PlatformConfigs {
		// 检查平台是否存在
		if _, ok := pm.registry.Get(platform); !ok {
			errors = append(errors, ValidationError{
				Platform: platform,
				Field:    "platform",
				Message:  fmt.Sprintf("unknown platform: %s", platform),
			})
		}

		// 验证速率限制
		if policy.RateLimit < 0 {
			errors = append(errors, ValidationError{
				Platform: platform,
				Field:    "rate_limit",
				Message:  "rate_limit must be non-negative",
			})
		}

		// 验证并发数
		if policy.Concurrency < 0 {
			errors = append(errors, ValidationError{
				Platform: platform,
				Field:    "concurrency",
				Message:  "concurrency must be non-negative",
			})
		}

		// 验证超时时间
		if policy.Timeout < 0 {
			errors = append(errors, ValidationError{
				Platform: platform,
				Field:    "timeout",
				Message:  "timeout must be non-negative",
			})
		}
	}

	return errors
}

// ValidationError 验证错误
type ValidationError struct {
	Platform string `json:"platform"`
	Field    string `json:"field"`
	Message  string `json:"message"`
}

// String 返回错误描述
func (e ValidationError) String() string {
	return fmt.Sprintf("[%s] %s: %s", e.Platform, e.Field, e.Message)
}
