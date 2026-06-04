package credentials

import (
	"os"
	"testing"
)

func TestDiscoverFromEnv(t *testing.T) {
	// 清理环境变量
	os.Unsetenv("SRC_HACKERONE")
	os.Unsetenv("SRC_HACKERONE_TOKEN")
	os.Unsetenv("SRC_HACKERONE_USERNAME")
	os.Unsetenv("SRC_BUGCROWD_APIKEY")

	config := DefaultDiscoveryConfig()
	discoverer := NewDiscoverer(config)

	// 测试格式1: SRC_PLATFORM=token:username
	os.Setenv("SRC_HACKERONE", "test-token:test-user")
	defer os.Unsetenv("SRC_HACKERONE")

	creds, err := discoverer.DiscoverFromEnv()
	if err != nil {
		t.Fatalf("DiscoverFromEnv failed: %v", err)
	}

	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}

	if creds[0].Platform != "hackerone" {
		t.Errorf("expected platform 'hackerone', got '%s'", creds[0].Platform)
	}
	if creds[0].Token != "test-token" {
		t.Errorf("expected token 'test-token', got '%s'", creds[0].Token)
	}
	if creds[0].Username != "test-user" {
		t.Errorf("expected username 'test-user', got '%s'", creds[0].Username)
	}
	if creds[0].Source != "env" {
		t.Errorf("expected source 'env', got '%s'", creds[0].Source)
	}
}

func TestDiscoverFromEnvSeparateVars(t *testing.T) {
	// 清理环境变量
	os.Unsetenv("SRC_HACKERONE")
	os.Unsetenv("SRC_HACKERONE_TOKEN")
	os.Unsetenv("SRC_HACKERONE_USERNAME")

	config := DefaultDiscoveryConfig()
	discoverer := NewDiscoverer(config)

	// 测试格式2: 分离的环境变量
	os.Setenv("SRC_HACKERONE_TOKEN", "my-token")
	os.Setenv("SRC_HACKERONE_USERNAME", "my-user")
	defer os.Unsetenv("SRC_HACKERONE_TOKEN")
	defer os.Unsetenv("SRC_HACKERONE_USERNAME")

	creds, err := discoverer.DiscoverFromEnv()
	if err != nil {
		t.Fatalf("DiscoverFromEnv failed: %v", err)
	}

	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}

	if creds[0].Token != "my-token" {
		t.Errorf("expected token 'my-token', got '%s'", creds[0].Token)
	}
	if creds[0].Username != "my-user" {
		t.Errorf("expected username 'my-user', got '%s'", creds[0].Username)
	}
}

func TestDiscoverFromEnvAPIKey(t *testing.T) {
	// 清理环境变量
	os.Unsetenv("SRC_BUGCROWD_APIKEY")

	config := DefaultDiscoveryConfig()
	discoverer := NewDiscoverer(config)

	// 测试 API Key
	os.Setenv("SRC_BUGCROWD_APIKEY", "my-api-key")
	defer os.Unsetenv("SRC_BUGCROWD_APIKEY")

	creds, err := discoverer.DiscoverFromEnv()
	if err != nil {
		t.Fatalf("DiscoverFromEnv failed: %v", err)
	}

	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}

	if creds[0].APIKey != "my-api-key" {
		t.Errorf("expected API key 'my-api-key', got '%s'", creds[0].APIKey)
	}
}

func TestDiscoverMultiplePlatforms(t *testing.T) {
	// 清理环境变量
	os.Unsetenv("SRC_HACKERONE")
	os.Unsetenv("SRC_BUGCROWD")

	config := DefaultDiscoveryConfig()
	discoverer := NewDiscoverer(config)

	// 设置多个平台
	os.Setenv("SRC_HACKERONE", "token1:user1")
	os.Setenv("SRC_BUGCROWD", "token2:user2")
	defer os.Unsetenv("SRC_HACKERONE")
	defer os.Unsetenv("SRC_BUGCROWD")

	creds, err := discoverer.DiscoverFromEnv()
	if err != nil {
		t.Fatalf("DiscoverFromEnv failed: %v", err)
	}

	if len(creds) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(creds))
	}
}

func TestHasCredential(t *testing.T) {
	// 清理环境变量
	os.Unsetenv("SRC_HACKERONE")

	config := DefaultDiscoveryConfig()
	discoverer := NewDiscoverer(config)

	// 没有凭证时
	if discoverer.HasCredential("hackerone") {
		t.Error("expected no credential for hackerone")
	}

	// 设置凭证
	os.Setenv("SRC_HACKERONE", "token:user")
	defer os.Unsetenv("SRC_HACKERONE")

	if !discoverer.HasCredential("hackerone") {
		t.Error("expected credential for hackerone")
	}
}

func TestListPlatforms(t *testing.T) {
	// 清理环境变量
	os.Unsetenv("SRC_HACKERONE")
	os.Unsetenv("SRC_BUGCROWD")

	config := DefaultDiscoveryConfig()
	discoverer := NewDiscoverer(config)

	// 没有凭证时
	platforms := discoverer.ListPlatforms()
	if len(platforms) != 0 {
		t.Errorf("expected 0 platforms, got %d", len(platforms))
	}

	// 设置凭证
	os.Setenv("SRC_HACKERONE", "token:user")
	os.Setenv("SRC_BUGCROWD", "token:user")
	defer os.Unsetenv("SRC_HACKERONE")
	defer os.Unsetenv("SRC_BUGCROWD")

	platforms = discoverer.ListPlatforms()
	if len(platforms) != 2 {
		t.Errorf("expected 2 platforms, got %d", len(platforms))
	}
}

func TestGetCredentialForPlatform(t *testing.T) {
	// 清理环境变量
	os.Unsetenv("SRC_HACKERONE")

	config := DefaultDiscoveryConfig()
	discoverer := NewDiscoverer(config)

	// 没有凭证时
	_, err := discoverer.GetCredentialForPlatform("hackerone")
	if err == nil {
		t.Error("expected error for missing credential")
	}

	// 设置凭证
	os.Setenv("SRC_HACKERONE", "token:user")
	defer os.Unsetenv("SRC_HACKERONE")

	cred, err := discoverer.GetCredentialForPlatform("hackerone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.Token != "token" {
		t.Errorf("expected token 'token', got '%s'", cred.Token)
	}
}
