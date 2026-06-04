package credentials

import (
	"os"
	"testing"
)

func TestDefaultPolicyConfig(t *testing.T) {
	config := DefaultPolicyConfig()

	// 检查默认平台配置
	if len(config.PlatformConfigs) == 0 {
		t.Fatal("expected platform configs")
	}

	// 检查 hackerone 配置
	hackerone, ok := config.PlatformConfigs["hackerone"]
	if !ok {
		t.Fatal("expected hackerone config")
	}
	if !hackerone.Enabled {
		t.Error("expected hackerone to be enabled")
	}
	if hackerone.RateLimit != 10 {
		t.Errorf("expected rate_limit 10, got %d", hackerone.RateLimit)
	}

	// 检查默认策略
	if config.DefaultPolicy.RateLimit != 5 {
		t.Errorf("expected default rate_limit 5, got %d", config.DefaultPolicy.RateLimit)
	}
}

func TestPolicyManagerGetPlatformPolicy(t *testing.T) {
	config := DefaultPolicyConfig()
	pm := NewPolicyManager(config)

	// 测试获取已配置平台的策略
	policy := pm.GetPlatformPolicy("hackerone")
	if policy.RateLimit != 10 {
		t.Errorf("expected rate_limit 10, got %d", policy.RateLimit)
	}

	// 测试获取未配置平台的策略（应返回默认策略）
	policy = pm.GetPlatformPolicy("unknown")
	if policy.RateLimit != 5 {
		t.Errorf("expected default rate_limit 5, got %d", policy.RateLimit)
	}
}

func TestPolicyManagerGetPlatformPolicyWithCredential(t *testing.T) {
	// 清理环境变量
	os.Unsetenv("SRC_HACKERONE")

	config := DefaultPolicyConfig()
	pm := NewPolicyManager(config)

	// 没有凭证时
	policyWithCred, err := pm.GetPlatformPolicyWithCredential("hackerone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policyWithCred.HasCredential {
		t.Error("expected no credential")
	}
	if policyWithCred.CredentialType != "none" {
		t.Errorf("expected credential_type 'none', got '%s'", policyWithCred.CredentialType)
	}

	// 设置凭证
	os.Setenv("SRC_HACKERONE", "token:user")
	defer os.Unsetenv("SRC_HACKERONE")

	// 重新创建 discoverer 以读取新环境变量
	pm.discoverer = NewDiscoverer(DefaultDiscoveryConfig())

	policyWithCred, err = pm.GetPlatformPolicyWithCredential("hackerone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !policyWithCred.HasCredential {
		t.Error("expected credential")
	}
	if policyWithCred.CredentialType != "session" {
		t.Errorf("expected credential_type 'session', got '%s'", policyWithCred.CredentialType)
	}
	if policyWithCred.Username != "user" {
		t.Errorf("expected username 'user', got '%s'", policyWithCred.Username)
	}
}

func TestPolicyManagerListEnabledPlatforms(t *testing.T) {
	config := DefaultPolicyConfig()
	pm := NewPolicyManager(config)

	platforms := pm.ListEnabledPlatforms()
	if len(platforms) == 0 {
		t.Fatal("expected at least one enabled platform")
	}

	// 检查是否包含 hackerone
	found := false
	for _, p := range platforms {
		if p == "hackerone" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected hackerone to be enabled")
	}
}

func TestPolicyManagerListEnabledPlatformsWithCredentials(t *testing.T) {
	// 清理环境变量
	os.Unsetenv("SRC_HACKERONE")
	os.Unsetenv("SRC_BUGCROWD")

	config := DefaultPolicyConfig()
	pm := NewPolicyManager(config)

	// 没有凭证时
	platforms := pm.ListEnabledPlatformsWithCredentials()
	if len(platforms) != 0 {
		t.Errorf("expected 0 platforms, got %d", len(platforms))
	}

	// 设置凭证
	os.Setenv("SRC_HACKERONE", "token:user")
	defer os.Unsetenv("SRC_HACKERONE")

	// 重新创建 discoverer
	pm.discoverer = NewDiscoverer(DefaultDiscoveryConfig())

	platforms = pm.ListEnabledPlatformsWithCredentials()
	if len(platforms) != 1 {
		t.Errorf("expected 1 platform, got %d", len(platforms))
	}
	if platforms[0] != "hackerone" {
		t.Errorf("expected 'hackerone', got '%s'", platforms[0])
	}
}

func TestPolicyManagerGetScanConfigForPlatform(t *testing.T) {
	// 清理环境变量
	os.Unsetenv("SRC_HACKERONE")

	config := DefaultPolicyConfig()
	pm := NewPolicyManager(config)

	// 没有凭证时
	scanConfig, err := pm.GetScanConfigForPlatform("hackerone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scanConfig.Platform != "hackerone" {
		t.Errorf("expected platform 'hackerone', got '%s'", scanConfig.Platform)
	}
	if scanConfig.HasCredential {
		t.Error("expected no credential")
	}
	if scanConfig.RateLimit != 10 {
		t.Errorf("expected rate_limit 10, got %d", scanConfig.RateLimit)
	}

	// 设置凭证
	os.Setenv("SRC_HACKERONE", "token:user")
	defer os.Unsetenv("SRC_HACKERONE")

	// 重新创建 discoverer
	pm.discoverer = NewDiscoverer(DefaultDiscoveryConfig())

	scanConfig, err = pm.GetScanConfigForPlatform("hackerone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !scanConfig.HasCredential {
		t.Error("expected credential")
	}
	if scanConfig.Username != "user" {
		t.Errorf("expected username 'user', got '%s'", scanConfig.Username)
	}
}

func TestPolicyManagerValidatePolicy(t *testing.T) {
	config := DefaultPolicyConfig()
	pm := NewPolicyManager(config)

	// 验证默认配置
	errors := pm.ValidatePolicy()
	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}

	// 测试无效配置
	config.PlatformConfigs["invalid"] = PlatformPolicy{
		Enabled:     true,
		RateLimit:   -1,
		Concurrency: -1,
		Timeout:     -1,
	}
	pm = NewPolicyManager(config)

	errors = pm.ValidatePolicy()
	// 应该有 4 个错误：1 个未知平台 + 3 个负值
	if len(errors) != 4 {
		t.Errorf("expected 4 errors, got %d", len(errors))
	}
}

func TestValidationErrorString(t *testing.T) {
	err := ValidationError{
		Platform: "hackerone",
		Field:    "rate_limit",
		Message:  "must be non-negative",
	}

	expected := "[hackerone] rate_limit: must be non-negative"
	if err.String() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.String())
	}
}
