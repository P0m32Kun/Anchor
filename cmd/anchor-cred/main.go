package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/credentials"
	"github.com/P0m32Kun/Anchor/internal/sources"
)

func main() {
	// 子命令
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "discover":
		handleDiscover()
	case "list":
		handleList()
	case "check":
		handleCheck()
	case "sources":
		handleSources()
	case "profiles":
		handleProfiles()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("anchor-cred - SRC platform credential discovery tool")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  anchor-cred <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  discover    Discover credentials from environment and browser")
	fmt.Println("  list        List all discovered credentials")
	fmt.Println("  check       Check if a platform has credentials")
	fmt.Println("  sources     List all supported SRC platforms")
	fmt.Println("  profiles    List browser profiles")
	fmt.Println("  help        Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  anchor-cred discover")
	fmt.Println("  anchor-cred list --format json")
	fmt.Println("  anchor-cred check --platform hackerone")
	fmt.Println("  anchor-cred sources --type bug_bounty")
	fmt.Println("  anchor-cred profiles")
}

func handleDiscover() {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	enableBrowser := fs.Bool("browser", false, "Enable browser cookie discovery")
	format := fs.String("format", "text", "Output format: text, json")
	fs.Parse(os.Args[2:])

	config := credentials.DefaultDiscoveryConfig()
	config.EnableBrowserDiscovery = *enableBrowser

	discoverer := credentials.NewDiscoverer(config)

	creds, err := discoverer.DiscoverAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering credentials: %v\n", err)
		os.Exit(1)
	}

	if *format == "json" {
		output := map[string]interface{}{
			"credentials": creds,
			"count":       len(creds),
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		if len(creds) == 0 {
			fmt.Println("No credentials found.")
			return
		}
		fmt.Printf("Found %d credential(s):\n\n", len(creds))
		for _, cred := range creds {
			fmt.Printf("Platform: %s\n", cred.Platform)
			if cred.Username != "" {
				fmt.Printf("  Username: %s\n", cred.Username)
			}
			if cred.Token != "" {
				fmt.Printf("  Token: %s...%s\n", cred.Token[:8], cred.Token[len(cred.Token)-4:])
			}
			if cred.APIKey != "" {
				fmt.Printf("  API Key: %s...%s\n", cred.APIKey[:8], cred.APIKey[len(cred.APIKey)-4:])
			}
			fmt.Printf("  Source: %s\n", cred.Source)
			fmt.Println()
		}
	}
}

func handleList() {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	format := fs.String("format", "text", "Output format: text, json")
	enableBrowser := fs.Bool("browser", false, "Include browser credentials")
	fs.Parse(os.Args[2:])

	config := credentials.DefaultDiscoveryConfig()
	config.EnableBrowserDiscovery = *enableBrowser

	discoverer := credentials.NewDiscoverer(config)

	creds, err := discoverer.DiscoverAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing credentials: %v\n", err)
		os.Exit(1)
	}

	if *format == "json" {
		type safeCredential struct {
			Platform string `json:"platform"`
			Username string `json:"username,omitempty"`
			HasAPI   bool   `json:"has_api"`
			HasToken bool   `json:"has_token"`
			Source   string `json:"source"`
		}

		result := make([]safeCredential, 0, len(creds))
		for _, cred := range creds {
			result = append(result, safeCredential{
				Platform: cred.Platform,
				Username: cred.Username,
				HasAPI:   cred.APIKey != "",
				HasToken: cred.Token != "",
				Source:   cred.Source,
			})
		}

		output := map[string]interface{}{
			"credentials": result,
			"count":       len(result),
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		if len(creds) == 0 {
			fmt.Println("No credentials found.")
			return
		}
		fmt.Printf("Found %d credential(s):\n\n", len(creds))
		for _, cred := range creds {
			fmt.Printf("  %-15s  %-20s  source=%s\n", cred.Platform, cred.Username, cred.Source)
		}
	}
}

func handleCheck() {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	platform := fs.String("platform", "", "Platform ID to check (required)")
	format := fs.String("format", "text", "Output format: text, json")
	enableBrowser := fs.Bool("browser", false, "Include browser credentials")
	fs.Parse(os.Args[2:])

	if *platform == "" {
		fmt.Fprintln(os.Stderr, "Error: --platform is required")
		os.Exit(1)
	}

	config := credentials.DefaultDiscoveryConfig()
	config.EnableBrowserDiscovery = *enableBrowser

	discoverer := credentials.NewDiscoverer(config)

	cred, err := discoverer.GetCredentialForPlatform(*platform)
	if err != nil {
		if *format == "json" {
			output := map[string]interface{}{
				"platform": *platform,
				"found":    false,
			}
			json.NewEncoder(os.Stdout).Encode(output)
		} else {
			fmt.Printf("No credential found for platform: %s\n", *platform)
		}
		return
	}

	if *format == "json" {
		output := map[string]interface{}{
			"platform": *platform,
			"found":    true,
			"username": cred.Username,
			"source":   cred.Source,
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Printf("Platform: %s\n", cred.Platform)
		if cred.Username != "" {
			fmt.Printf("  Username: %s\n", cred.Username)
		}
		fmt.Printf("  Source: %s\n", cred.Source)
	}
}

func handleSources() {
	fs := flag.NewFlagSet("sources", flag.ExitOnError)
	platformType := fs.String("type", "", "Filter by type: bug_bounty, src, internal")
	format := fs.String("format", "text", "Output format: text, json")
	fs.Parse(os.Args[2:])

	registry := sources.NewRegistry()

	var platforms []*sources.Platform
	if *platformType != "" {
		platforms = registry.ListByType(sources.PlatformType(*platformType))
	} else {
		platforms = registry.List()
	}

	if *format == "json" {
		output := map[string]interface{}{
			"sources": platforms,
			"count":   len(platforms),
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		if len(platforms) == 0 {
			fmt.Println("No sources found.")
			return
		}
		fmt.Printf("Found %d source(s):\n\n", len(platforms))
		for _, p := range platforms {
			fmt.Printf("  %-15s  %-20s  type=%-12s  api=%-5v  session=%v\n",
				p.ID, p.Name, p.Type, p.HasAPI, p.HasSession)
			if len(p.Domains) > 0 {
				fmt.Printf("                   domains: %s\n", strings.Join(p.Domains, ", "))
			}
		}
	}
}

func handleProfiles() {
	fs := flag.NewFlagSet("profiles", flag.ExitOnError)
	format := fs.String("format", "text", "Output format: text, json")
	fs.Parse(os.Args[2:])

	profiles, err := credentials.GetBrowserProfiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting browser profiles: %v\n", err)
		os.Exit(1)
	}

	if *format == "json" {
		output := map[string]interface{}{
			"profiles": profiles,
			"count":    len(profiles),
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		if len(profiles) == 0 {
			fmt.Println("No browser profiles found.")
			return
		}
		fmt.Printf("Found %d browser profile(s):\n\n", len(profiles))
		for _, p := range profiles {
			fmt.Printf("  %-30s  browser=%-10s  platform=%s\n", p.Name, p.Browser, p.Platform)
			fmt.Printf("    path: %s\n", p.Path)
		}
	}
}
