package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// BrowserCookieEntry 浏览器 cookie 条目
type BrowserCookieEntry struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Expires  int64  `json:"expires"`
	HTTPOnly bool   `json:"httpOnly"`
	Secure   bool   `json:"secure"`
}

// BrowserProfile 浏览器配置文件
type BrowserProfile struct {
	Name     string
	Path     string
	Browser  string // "chrome", "firefox", "edge", "brave"
	Platform string // macOS, linux, windows
}

// BrowserDiscoverer 浏览器凭证发现器
type BrowserDiscoverer struct {
	cookiePaths []string
}

// NewBrowserDiscoverer 创建浏览器凭证发现器
func NewBrowserDiscoverer(cookiePaths []string) *BrowserDiscoverer {
	return &BrowserDiscoverer{
		cookiePaths: cookiePaths,
	}
}

// DiscoverFromBrowser 从浏览器 cookie 发现凭证
func (d *BrowserDiscoverer) DiscoverFromBrowser() ([]*Credential, error) {
	var credentials []*Credential

	// 获取默认浏览器 cookie 路径
	if len(d.cookiePaths) == 0 {
		paths, err := getDefaultCookiePaths()
		if err != nil {
			return nil, fmt.Errorf("get default cookie paths: %w", err)
		}
		d.cookiePaths = paths
	}

	// 遍历所有 cookie 路径
	for _, path := range d.cookiePaths {
		creds, err := d.extractCredentialsFromCookieFile(path)
		if err != nil {
			// 单个文件失败不阻断
			fmt.Printf("[browser] failed to read %s: %v\n", path, err)
			continue
		}
		credentials = append(credentials, creds...)
	}

	return credentials, nil
}

// getDefaultCookiePaths 获取默认浏览器 cookie 路径
func getDefaultCookiePaths() ([]string, error) {
	var paths []string

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	switch runtime.GOOS {
	case "darwin":
		// macOS
		paths = append(paths,
			filepath.Join(home, "Library/Application Support/Google/Chrome/Default/Cookies"),
			filepath.Join(home, "Library/Application Support/Google/Chrome/Profile 1/Cookies"),
			filepath.Join(home, "Library/Application Support/Firefox/Profiles/*/cookies.sqlite"),
			filepath.Join(home, "Library/Application Support/Microsoft Edge/Default/Cookies"),
			filepath.Join(home, "Library/Application Support/BraveSoftware/Brave-Browser/Default/Cookies"),
		)
	case "linux":
		// Linux
		paths = append(paths,
			filepath.Join(home, ".config/google-chrome/Default/Cookies"),
			filepath.Join(home, ".config/google-chrome/Profile 1/Cookies"),
			filepath.Join(home, ".mozilla/firefox/*/cookies.sqlite"),
			filepath.Join(home, ".config/microsoft-edge/Default/Cookies"),
			filepath.Join(home, ".config/BraveSoftware/Brave-Browser/Default/Cookies"),
		)
	case "windows":
		// Windows
		localAppData := os.Getenv("LOCALAPPDATA")
		appData := os.Getenv("APPDATA")
		if localAppData != "" {
			paths = append(paths,
				filepath.Join(localAppData, "Google/Chrome/User Data/Default/Cookies"),
				filepath.Join(localAppData, "Google/Chrome/User Data/Profile 1/Cookies"),
				filepath.Join(localAppData, "Microsoft/Edge/User Data/Default/Cookies"),
				filepath.Join(localAppData, "BraveSoftware/Brave-Browser/User Data/Default/Cookies"),
			)
		}
		if appData != "" {
			paths = append(paths,
				filepath.Join(appData, "Mozilla/Firefox/Profiles/*/cookies.sqlite"),
			)
		}
	}

	return paths, nil
}

// extractCredentialsFromCookieFile 从 cookie 文件提取凭证
func (d *BrowserDiscoverer) extractCredentialsFromCookieFile(path string) ([]*Credential, error) {
	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	// 根据文件扩展名选择解析器
	ext := filepath.Ext(path)
	switch ext {
	case ".sqlite":
		return d.parseSQLiteCookies(path)
	case ".json":
		return d.parseJSONCookies(path)
	default:
		// 尝试作为 JSON 解析
		return d.parseJSONCookies(path)
	}
}

// parseSQLiteCookies 解析 SQLite cookie 文件
func (d *BrowserDiscoverer) parseSQLiteCookies(path string) ([]*Credential, error) {
	// SQLite 解析需要依赖数据库驱动
	// 这里预留接口，实际实现需要添加 sqlite 依赖
	return nil, fmt.Errorf("sqlite cookie parsing not implemented yet")
}

// parseJSONCookies 解析 JSON cookie 文件
func (d *BrowserDiscoverer) parseJSONCookies(path string) ([]*Credential, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var cookies []BrowserCookieEntry
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	var credentials []*Credential

	// 平台域名映射
	platformDomains := map[string]string{
		"hackerone.com":  "hackerone",
		"bugcrowd.com":   "bugcrowd",
		"intigriti.com":  "intigriti",
		"yeswehack.com":  "yeswehack",
		"disijie.com":    "disijie",
		"qianxin.com":    "qianxin",
		"topsec.com":     "topsec",
	}

	// 关键 cookie 名称映射
	cookieNames := map[string]bool{
		"session":        true,
		"token":          true,
		"auth_token":     true,
		"access_token":   true,
		"csrf_token":     true,
		"_session":       true,
		"remember_token": true,
	}

	for _, cookie := range cookies {
		// 检查是否是平台域名
		platform := ""
		for domain, p := range platformDomains {
			if strings.Contains(cookie.Domain, domain) {
				platform = p
				break
			}
		}

		if platform == "" {
			continue
		}

		// 检查是否是关键 cookie
		if !cookieNames[cookie.Name] {
			continue
		}

		// 创建凭证
		cred := &Credential{
			Platform: platform,
			Token:    cookie.Value,
			Source:   "browser",
		}

		// 尝试从 cookie 中提取用户名
		if cookie.Name == "session" || cookie.Name == "_session" {
			// session cookie 可能包含用户信息
			// 这里简化处理，实际可能需要解析 JWT 或其他格式
		}

		credentials = append(credentials, cred)
	}

	return credentials, nil
}

// GetBrowserProfiles 获取浏览器配置文件列表
func GetBrowserProfiles() ([]BrowserProfile, error) {
	var profiles []BrowserProfile

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	switch runtime.GOOS {
	case "darwin":
		// Chrome
		chromePath := filepath.Join(home, "Library/Application Support/Google/Chrome")
		if _, err := os.Stat(chromePath); err == nil {
			entries, _ := os.ReadDir(chromePath)
			for _, entry := range entries {
				if entry.IsDir() && (entry.Name() == "Default" || strings.HasPrefix(entry.Name(), "Profile ")) {
					profiles = append(profiles, BrowserProfile{
						Name:     "Chrome - " + entry.Name(),
						Path:     filepath.Join(chromePath, entry.Name(), "Cookies"),
						Browser:  "chrome",
						Platform: "darwin",
					})
				}
			}
		}

		// Firefox
		firefoxPath := filepath.Join(home, "Library/Application Support/Firefox/Profiles")
		if _, err := os.Stat(firefoxPath); err == nil {
			entries, _ := os.ReadDir(firefoxPath)
			for _, entry := range entries {
				if entry.IsDir() {
					profiles = append(profiles, BrowserProfile{
						Name:     "Firefox - " + entry.Name(),
						Path:     filepath.Join(firefoxPath, entry.Name(), "cookies.sqlite"),
						Browser:  "firefox",
						Platform: "darwin",
					})
				}
			}
		}

		// Edge
		edgePath := filepath.Join(home, "Library/Application Support/Microsoft Edge")
		if _, err := os.Stat(edgePath); err == nil {
			entries, _ := os.ReadDir(edgePath)
			for _, entry := range entries {
				if entry.IsDir() && (entry.Name() == "Default" || strings.HasPrefix(entry.Name(), "Profile ")) {
					profiles = append(profiles, BrowserProfile{
						Name:     "Edge - " + entry.Name(),
						Path:     filepath.Join(edgePath, entry.Name(), "Cookies"),
						Browser:  "edge",
						Platform: "darwin",
					})
				}
			}
		}

		// Brave
		bravePath := filepath.Join(home, "Library/Application Support/BraveSoftware/Brave-Browser")
		if _, err := os.Stat(bravePath); err == nil {
			entries, _ := os.ReadDir(bravePath)
			for _, entry := range entries {
				if entry.IsDir() && (entry.Name() == "Default" || strings.HasPrefix(entry.Name(), "Profile ")) {
					profiles = append(profiles, BrowserProfile{
						Name:     "Brave - " + entry.Name(),
						Path:     filepath.Join(bravePath, entry.Name(), "Cookies"),
						Browser:  "brave",
						Platform: "darwin",
					})
				}
			}
		}
	}

	return profiles, nil
}
