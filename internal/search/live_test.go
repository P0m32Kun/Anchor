package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// TestLiveHunterSearch 调用真实 Hunter API 测试解析逻辑
// 运行方式: HUNTER_API_KEY=xxx go test -v -run TestLiveHunterSearch ./internal/search/
func TestLiveHunterSearch(t *testing.T) {
	apiKey := os.Getenv("HUNTER_API_KEY")
	if apiKey == "" {
		t.Skip("HUNTER_API_KEY not set, skipping live test")
	}

	client := NewHunterClient(apiKey)
	results, err := client.Search(context.Background(), "domain=\"baidu.com\"", 1, 10)
	if err != nil {
		t.Fatalf("Hunter search failed: %v", err)
	}

	fmt.Println("\n========== Hunter 真实 API 测试结果 ==========")
	fmt.Printf("共返回 %d 条结果\n\n", len(results))
	for i, r := range results {
		if i >= 3 {
			fmt.Printf("... 还有 %d 条结果\n", len(results)-3)
			break
		}
		fmt.Printf("--- 结果 #%d ---\n", i+1)
		fmt.Printf("  IP:           %s\n", r.IP)
		fmt.Printf("  Port:         %d\n", r.Port)
		fmt.Printf("  Domain:       %s\n", r.Domain)
		fmt.Printf("  Title:        %s\n", r.Title)
		fmt.Printf("  Service:      %s\n", r.Service)
		fmt.Printf("  Protocol:     %s\n", r.Protocol)
		fmt.Printf("  Organization: %s\n", r.Organization)
		fmt.Printf("  ICP:          %s\n", r.ICP)
		fmt.Printf("  StatusCode:   %d\n", r.StatusCode)
		fmt.Printf("  OS:           %s\n", r.OS)
		fmt.Printf("  Location:     %s\n", r.Location)
		fmt.Println()
	}

	// 打印第一条原始数据用于调试
	if len(results) > 0 {
		raw, _ := json.MarshalIndent(results[0].Raw, "", "  ")
		fmt.Println("========== 第一条原始数据 ==========")
		fmt.Println(string(raw))
	}
}

// TestLiveQuakeSearch 调用真实 Quake API 测试解析逻辑
// 运行方式: QUAKE_API_KEY=xxx go test -v -run TestLiveQuakeSearch ./internal/search/
func TestLiveQuakeSearch(t *testing.T) {
	apiKey := os.Getenv("QUAKE_API_KEY")
	if apiKey == "" {
		t.Skip("QUAKE_API_KEY not set, skipping live test")
	}

	client := NewQuakeClient(apiKey)
	results, err := client.Search(context.Background(), "domain:baidu.com", 0, 5)
	if err != nil {
		t.Fatalf("Quake search failed: %v", err)
	}

	fmt.Println("\n========== Quake 真实 API 测试结果 ==========")
	fmt.Printf("共返回 %d 条结果\n\n", len(results))
	for i, r := range results {
		if i >= 3 {
			fmt.Printf("... 还有 %d 条结果\n", len(results)-3)
			break
		}
		fmt.Printf("--- 结果 #%d ---\n", i+1)
		fmt.Printf("  IP:           %s\n", r.IP)
		fmt.Printf("  Port:         %d\n", r.Port)
		fmt.Printf("  Domain:       %s\n", r.Domain)
		fmt.Printf("  Title:        %s\n", r.Title)
		fmt.Printf("  Service:      %s\n", r.Service)
		fmt.Printf("  Protocol:     %s\n", r.Protocol)
		fmt.Printf("  Organization: %s\n", r.Organization)
		fmt.Printf("  StatusCode:   %d\n", r.StatusCode)
		fmt.Printf("  OS:           %s\n", r.OS)
		fmt.Printf("  Location:     %s\n", r.Location)
		fmt.Println()
	}

	// 打印第一条原始数据用于调试
	if len(results) > 0 {
		raw, _ := json.MarshalIndent(results[0].Raw, "", "  ")
		fmt.Println("========== 第一条原始数据 ==========")
		fmt.Println(string(raw))
	}
}

// TestLiveHunterQuota 测试 Hunter 额度查询
// 运行方式: HUNTER_API_KEY=xxx go test -v -run TestLiveHunterQuota ./internal/search/
func TestLiveHunterQuota(t *testing.T) {
	apiKey := os.Getenv("HUNTER_API_KEY")
	if apiKey == "" {
		t.Skip("HUNTER_API_KEY not set, skipping live test")
	}

	client := NewHunterClient(apiKey)
	quota, err := client.GetQuota(context.Background())
	if err != nil {
		t.Fatalf("Hunter quota failed: %v", err)
	}

	fmt.Println("\n========== Hunter 额度查询结果 ==========")
	for _, p := range quota.Points {
		fmt.Printf("  %s: %d %s\n", p.Name, p.Value, p.Unit)
	}
}

// TestLiveQuakeQuota 测试 Quake 额度查询
// 运行方式: QUAKE_API_KEY=xxx go test -v -run TestLiveQuakeQuota ./internal/search/
func TestLiveQuakeQuota(t *testing.T) {
	apiKey := os.Getenv("QUAKE_API_KEY")
	if apiKey == "" {
		t.Skip("QUAKE_API_KEY not set, skipping live test")
	}

	client := NewQuakeClient(apiKey)
	quota, err := client.GetQuota(context.Background())
	if err != nil {
		t.Fatalf("Quake quota failed: %v", err)
	}

	fmt.Println("\n========== Quake 额度查询结果 ==========")
	for _, p := range quota.Points {
		fmt.Printf("  %s: %d %s\n", p.Name, p.Value, p.Unit)
	}
}
