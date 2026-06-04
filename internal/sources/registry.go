package sources

import (
	"fmt"
	"strings"
)

// PlatformType 平台类型
type PlatformType string

const (
	PlatformTypeBugBounty PlatformType = "bug_bounty" // 漏洞赏金平台
	PlatformTypeSRC       PlatformType = "src"        // 企业 SRC
	PlatformTypeInternal  PlatformType = "internal"   // 内部平台
)

// Platform 平台元数据
type Platform struct {
	ID          string       // 唯一标识，如 "hackerone", "bugcrowd"
	Name        string       // 显示名称
	Type        PlatformType // 平台类型
	Domains     []string     // 关联域名
	HasAPI      bool         // 是否有 API
	HasSession  bool         // 是否支持 session token
	EnvVarKey   string       // 环境变量名（不含前缀）
	Description string       // 描述
}

// Registry 平台注册表
type Registry struct {
	platforms map[string]*Platform
}

// NewRegistry 创建平台注册表
func NewRegistry() *Registry {
	r := &Registry{
		platforms: make(map[string]*Platform),
	}
	r.registerDefaults()
	return r
}

// registerDefaults 注册默认平台
func (r *Registry) registerDefaults() {
	// 漏洞赏金平台
	r.Register(&Platform{
		ID:         "hackerone",
		Name:       "HackerOne",
		Type:       PlatformTypeBugBounty,
		Domains:    []string{"hackerone.com", "hackerone.net"},
		HasAPI:     true,
		HasSession: true,
		EnvVarKey:  "HACKERONE",
	})

	r.Register(&Platform{
		ID:         "bugcrowd",
		Name:       "Bugcrowd",
		Type:       PlatformTypeBugBounty,
		Domains:    []string{"bugcrowd.com", "bugcrowd.net"},
		HasAPI:     true,
		HasSession: true,
		EnvVarKey:  "BUGCROWD",
	})

	r.Register(&Platform{
		ID:         "intigriti",
		Name:       "Intigriti",
		Type:       PlatformTypeBugBounty,
		Domains:    []string{"intigriti.com"},
		HasAPI:     true,
		HasSession: false,
		EnvVarKey:  "INTIGRITI",
	})

	r.Register(&Platform{
		ID:         "yeswehack",
		Name:       "YesWeHack",
		Type:       PlatformTypeBugBounty,
		Domains:    []string{"yeswehack.com"},
		HasAPI:     true,
		HasSession: false,
		EnvVarKey:  "YESWEHACK",
	})

	// 企业 SRC 平台
	r.Register(&Platform{
		ID:         "disijie",
		Name:       "迪思杰",
		Type:       PlatformTypeSRC,
		Domains:    []string{"disijie.com"},
		HasAPI:     false,
		HasSession: true,
		EnvVarKey:  "DISIJIE",
	})

	r.Register(&Platform{
		ID:         "qianxin",
		Name:       "奇安信",
		Type:       PlatformTypeSRC,
		Domains:    []string{"qianxin.com", "qianxin-inc.com"},
		HasAPI:     false,
		HasSession: true,
		EnvVarKey:  "QIANXIN",
	})

	r.Register(&Platform{
		ID:         "topsec",
		Name:       "天融信",
		Type:       PlatformTypeSRC,
		Domains:    []string{"topsec.com", "topsec.com.cn"},
		HasAPI:     false,
		HasSession: true,
		EnvVarKey:  "TOPSEC",
	})

	r.Register(&Platform{
		ID:         "dbappsecurity",
		Name:       "安恒信息",
		Type:       PlatformTypeSRC,
		Domains:    []string{"dbappsecurity.com.cn"},
		HasAPI:     false,
		HasSession: true,
		EnvVarKey:  "DBAPPSECURITY",
	})
}

// Register 注册平台
func (r *Registry) Register(platform *Platform) {
	r.platforms[strings.ToLower(platform.ID)] = platform
}

// Get 获取平台
func (r *Registry) Get(id string) (*Platform, bool) {
	p, ok := r.platforms[strings.ToLower(id)]
	return p, ok
}

// GetByDomain 通过域名查找平台
func (r *Registry) GetByDomain(domain string) *Platform {
	domain = strings.ToLower(domain)
	for _, p := range r.platforms {
		for _, d := range p.Domains {
			if domain == d || strings.HasSuffix(domain, "."+d) {
				return p
			}
		}
	}
	return nil
}

// GetByEnvVar 通过环境变量名查找平台
func (r *Registry) GetByEnvVar(envVar string) *Platform {
	envVar = strings.ToUpper(envVar)
	for _, p := range r.platforms {
		if strings.ToUpper(p.EnvVarKey) == envVar {
			return p
		}
	}
	return nil
}

// List 列出所有平台
func (r *Registry) List() []*Platform {
	platforms := make([]*Platform, 0, len(r.platforms))
	for _, p := range r.platforms {
		platforms = append(platforms, p)
	}
	return platforms
}

// ListByType 按类型列出平台
func (r *Registry) ListByType(platformType PlatformType) []*Platform {
	var platforms []*Platform
	for _, p := range r.platforms {
		if p.Type == platformType {
			platforms = append(platforms, p)
		}
	}
	return platforms
}

// ListByDomain 列出与域名相关的所有平台
func (r *Registry) ListByDomain(domain string) []*Platform {
	domain = strings.ToLower(domain)
	var platforms []*Platform
	for _, p := range r.platforms {
		for _, d := range p.Domains {
			if domain == d || strings.HasSuffix(domain, "."+d) {
				platforms = append(platforms, p)
				break
			}
		}
	}
	return platforms
}

// HasAPI 检查平台是否有 API
func (r *Registry) HasAPI(platformID string) bool {
	p, ok := r.Get(platformID)
	if !ok {
		return false
	}
	return p.HasAPI
}

// HasSession 检查平台是否支持 session token
func (r *Registry) HasSession(platformID string) bool {
	p, ok := r.Get(platformID)
	if !ok {
		return false
	}
	return p.HasSession
}

// GetDomains 获取所有平台关联的域名
func (r *Registry) GetDomains() []string {
	domains := make([]string, 0)
	seen := make(map[string]bool)
	for _, p := range r.platforms {
		for _, d := range p.Domains {
			if !seen[d] {
				domains = append(domains, d)
				seen[d] = true
			}
		}
	}
	return domains
}

// IsPlatformDomain 检查域名是否是平台域名
func (r *Registry) IsPlatformDomain(domain string) bool {
	return r.GetByDomain(domain) != nil
}

// FormatCredentialEnvVar 格式化凭证环境变量名
func FormatCredentialEnvVar(platformID, field string) string {
	return fmt.Sprintf("SRC_%s_%s", strings.ToUpper(platformID), strings.ToUpper(field))
}
