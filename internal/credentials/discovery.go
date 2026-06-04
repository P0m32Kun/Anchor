package credentials

import (
	"fmt"
	"os"
	"strings"
)

// Credential 表示一个已发现的平台凭证
type Credential struct {
	Platform string // 平台标识，如 "hackerone", "bugcrowd"
	Username string // 账号用户名
	APIKey   string // API Key（可选）
	Token    string // Session Token（可选）
	Source   string // 来源："env", "browser"
}

// DiscoveryConfig 凭证发现配置
type DiscoveryConfig struct {
	// 环境变量前缀，如 "SRC_"
	EnvPrefix string
	// 是否尝试浏览器发现
	EnableBrowserDiscovery bool
	// 浏览器 cookie 文件路径（可选）
	BrowserCookiePaths []string
}

// DefaultDiscoveryConfig 返回默认配置
func DefaultDiscoveryConfig() DiscoveryConfig {
	return DiscoveryConfig{
		EnvPrefix:              "SRC_",
		EnableBrowserDiscovery: false,
	}
}

// Discoverer 凭证发现器
type Discoverer struct {
	config DiscoveryConfig
}

// NewDiscoverer 创建凭证发现器
func NewDiscoverer(config DiscoveryConfig) *Discoverer {
	return &Discoverer{config: config}
}

// DiscoverAll 执行全量凭证发现
func (d *Discoverer) DiscoverAll() ([]*Credential, error) {
	var credentials []*Credential

	// 1. 环境变量发现
	envCreds, err := d.DiscoverFromEnv()
	if err != nil {
		return nil, fmt.Errorf("env discovery: %w", err)
	}
	credentials = append(credentials, envCreds...)

	// 2. 浏览器发现（如果启用）
	if d.config.EnableBrowserDiscovery {
		browserCreds, err := d.DiscoverFromBrowser()
		if err != nil {
			// 浏览器发现失败不阻断，只记录
			fmt.Printf("[credentials] browser discovery failed: %v\n", err)
		} else {
			credentials = append(credentials, browserCreds...)
		}
	}

	return credentials, nil
}

// DiscoverFromEnv 从环境变量发现凭证
// 支持两种格式：
//   - SRC_HACKERONE=token:username
//   - SRC_HACKERONE_TOKEN=token
//   - SRC_HACKERONE_USERNAME=username
func (d *Discoverer) DiscoverFromEnv() ([]*Credential, error) {
	var credentials []*Credential
	prefix := d.config.EnvPrefix

	// 收集所有 SRC_* 环境变量
	envVars := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		if strings.HasPrefix(key, prefix) {
			envVars[key] = value
		}
	}

	// 按平台分组
	platformCreds := make(map[string]*Credential)

	for key, value := range envVars {
		// 去掉前缀
		suffix := strings.TrimPrefix(key, prefix)
		suffixLower := strings.ToLower(suffix)

		// 格式1: SRC_PLATFORM=token:username
		if !strings.Contains(suffixLower, "_") && strings.Contains(value, ":") {
			parts := strings.SplitN(value, ":", 2)
			if len(parts) == 2 {
				platform := strings.ToLower(suffix)
				if _, exists := platformCreds[platform]; !exists {
					platformCreds[platform] = &Credential{
						Platform: platform,
						Source:   "env",
					}
				}
				platformCreds[platform].Token = parts[0]
				platformCreds[platform].Username = parts[1]
			}
			continue
		}

		// 格式2: SRC_PLATFORM_TOKEN, SRC_PLATFORM_USERNAME, SRC_PLATFORM_APIKEY
		parts := strings.SplitN(suffixLower, "_", 2)
		if len(parts) == 2 {
			platform := parts[0]
			field := parts[1]

			if _, exists := platformCreds[platform]; !exists {
				platformCreds[platform] = &Credential{
					Platform: platform,
					Source:   "env",
				}
			}

			switch field {
			case "token":
				platformCreds[platform].Token = value
			case "username":
				platformCreds[platform].Username = value
			case "apikey", "api_key", "key":
				platformCreds[platform].APIKey = value
			}
		}
	}

	// 转换为切片
	for _, cred := range platformCreds {
		if cred.Token != "" || cred.APIKey != "" {
			credentials = append(credentials, cred)
		}
	}

	return credentials, nil
}

// DiscoverFromBrowser 从浏览器 cookie 发现凭证
func (d *Discoverer) DiscoverFromBrowser() ([]*Credential, error) {
	browserDiscoverer := NewBrowserDiscoverer(d.config.BrowserCookiePaths)
	return browserDiscoverer.DiscoverFromBrowser()
}

// GetCredentialForPlatform 获取指定平台的凭证
func (d *Discoverer) GetCredentialForPlatform(platform string) (*Credential, error) {
	creds, err := d.DiscoverAll()
	if err != nil {
		return nil, err
	}

	for _, cred := range creds {
		if strings.EqualFold(cred.Platform, platform) {
			return cred, nil
		}
	}

	return nil, fmt.Errorf("no credential found for platform: %s", platform)
}

// HasCredential 检查是否有指定平台的凭证
func (d *Discoverer) HasCredential(platform string) bool {
	cred, _ := d.GetCredentialForPlatform(platform)
	return cred != nil
}

// ListPlatforms 列出所有已发现凭证的平台
func (d *Discoverer) ListPlatforms() []string {
	creds, err := d.DiscoverAll()
	if err != nil {
		return nil
	}

	platforms := make([]string, 0, len(creds))
	for _, cred := range creds {
		platforms = append(platforms, cred.Platform)
	}
	return platforms
}
