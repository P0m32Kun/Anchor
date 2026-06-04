package sources

import (
	"testing"
)

func TestRegistryGet(t *testing.T) {
	registry := NewRegistry()

	// 测试获取存在的平台
	platform, ok := registry.Get("hackerone")
	if !ok {
		t.Fatal("expected to find hackerone")
	}
	if platform.Name != "HackerOne" {
		t.Errorf("expected name 'HackerOne', got '%s'", platform.Name)
	}
	if platform.Type != PlatformTypeBugBounty {
		t.Errorf("expected type 'bug_bounty', got '%s'", platform.Type)
	}

	// 测试获取不存在的平台
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent platform")
	}

	// 测试大小写不敏感
	_, ok = registry.Get("HackerOne")
	if !ok {
		t.Error("expected case-insensitive lookup")
	}
}

func TestRegistryGetByDomain(t *testing.T) {
	registry := NewRegistry()

	// 测试精确域名匹配
	platform := registry.GetByDomain("hackerone.com")
	if platform == nil {
		t.Fatal("expected to find platform for hackerone.com")
	}
	if platform.ID != "hackerone" {
		t.Errorf("expected platform 'hackerone', got '%s'", platform.ID)
	}

	// 测试子域名匹配
	platform = registry.GetByDomain("api.hackerone.com")
	if platform == nil {
		t.Fatal("expected to find platform for api.hackerone.com")
	}
	if platform.ID != "hackerone" {
		t.Errorf("expected platform 'hackerone', got '%s'", platform.ID)
	}

	// 测试不存在的域名
	platform = registry.GetByDomain("example.com")
	if platform != nil {
		t.Error("expected not to find platform for example.com")
	}
}

func TestRegistryGetByEnvVar(t *testing.T) {
	registry := NewRegistry()

	// 测试通过环境变量名查找
	platform := registry.GetByEnvVar("HACKERONE")
	if platform == nil {
		t.Fatal("expected to find platform for HACKERONE")
	}
	if platform.ID != "hackerone" {
		t.Errorf("expected platform 'hackerone', got '%s'", platform.ID)
	}

	// 测试大小写不敏感
	platform = registry.GetByEnvVar("hackerone")
	if platform == nil {
		t.Fatal("expected case-insensitive lookup")
	}
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()

	platforms := registry.List()
	if len(platforms) == 0 {
		t.Fatal("expected at least one platform")
	}

	// 检查是否包含默认平台
	found := false
	for _, p := range platforms {
		if p.ID == "hackerone" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find hackerone in list")
	}
}

func TestRegistryListByType(t *testing.T) {
	registry := NewRegistry()

	// 测试按类型筛选
	bugBountyPlatforms := registry.ListByType(PlatformTypeBugBounty)
	if len(bugBountyPlatforms) == 0 {
		t.Fatal("expected at least one bug bounty platform")
	}

	for _, p := range bugBountyPlatforms {
		if p.Type != PlatformTypeBugBounty {
			t.Errorf("expected type 'bug_bounty', got '%s'", p.Type)
		}
	}

	srcPlatforms := registry.ListByType(PlatformTypeSRC)
	if len(srcPlatforms) == 0 {
		t.Fatal("expected at least one SRC platform")
	}
}

func TestRegistryListByDomain(t *testing.T) {
	registry := NewRegistry()

	// 测试列出与域名相关的平台
	platforms := registry.ListByDomain("hackerone.com")
	if len(platforms) == 0 {
		t.Fatal("expected at least one platform for hackerone.com")
	}

	found := false
	for _, p := range platforms {
		if p.ID == "hackerone" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find hackerone for hackerone.com")
	}
}

func TestRegistryHasAPI(t *testing.T) {
	registry := NewRegistry()

	// hackerone 有 API
	if !registry.HasAPI("hackerone") {
		t.Error("expected hackerone to have API")
	}

	// 不存在的平台
	if registry.HasAPI("nonexistent") {
		t.Error("expected nonexistent platform to not have API")
	}
}

func TestRegistryHasSession(t *testing.T) {
	registry := NewRegistry()

	// hackerone 支持 session
	if !registry.HasSession("hackerone") {
		t.Error("expected hackerone to support session")
	}

	// intigriti 不支持 session
	if registry.HasSession("intigriti") {
		t.Error("expected intigriti to not support session")
	}
}

func TestRegistryGetDomains(t *testing.T) {
	registry := NewRegistry()

	domains := registry.GetDomains()
	if len(domains) == 0 {
		t.Fatal("expected at least one domain")
	}

	// 检查是否包含 hackerone.com
	found := false
	for _, d := range domains {
		if d == "hackerone.com" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find hackerone.com in domains")
	}
}

func TestRegistryIsPlatformDomain(t *testing.T) {
	registry := NewRegistry()

	// 测试平台域名
	if !registry.IsPlatformDomain("hackerone.com") {
		t.Error("expected hackerone.com to be platform domain")
	}

	// 测试子域名
	if !registry.IsPlatformDomain("api.hackerone.com") {
		t.Error("expected api.hackerone.com to be platform domain")
	}

	// 测试非平台域名
	if registry.IsPlatformDomain("example.com") {
		t.Error("expected example.com to not be platform domain")
	}
}

func TestFormatCredentialEnvVar(t *testing.T) {
	// 测试格式化环境变量名
	result := FormatCredentialEnvVar("hackerone", "token")
	expected := "SRC_HACKERONE_TOKEN"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}

	result = FormatCredentialEnvVar("bugcrowd", "username")
	expected = "SRC_BUGCROWD_USERNAME"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	// 注册新平台
	registry.Register(&Platform{
		ID:         "testplatform",
		Name:       "Test Platform",
		Type:       PlatformTypeBugBounty,
		Domains:    []string{"test.com"},
		HasAPI:     true,
		HasSession: true,
		EnvVarKey:  "TEST",
	})

	// 验证注册成功
	platform, ok := registry.Get("testplatform")
	if !ok {
		t.Fatal("expected to find testplatform")
	}
	if platform.Name != "Test Platform" {
		t.Errorf("expected name 'Test Platform', got '%s'", platform.Name)
	}
}
