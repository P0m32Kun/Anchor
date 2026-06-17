package scanconfig

import (
	"bufio"
	"os"
	"strings"
)

func readLinesFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, scanner.Err()
}

func readPortsFile(path string) (string, error) {
	lines, err := readLinesFile(path)
	if err != nil {
		return "", err
	}
	var parts []string
	for _, line := range lines {
		for _, p := range strings.Split(line, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				parts = append(parts, p)
			}
		}
	}
	return strings.Join(parts, ","), nil
}

func readJunkKeywordsFile(path string) ([]JunkKeyword, error) {
	lines, err := readLinesFile(path)
	if err != nil {
		return nil, err
	}
	var out []JunkKeyword
	for _, line := range lines {
		wb := false
		if idx := strings.Index(strings.ToLower(line), " word_boundary"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
			wb = true
		}
		if line == "" {
			continue
		}
		out = append(out, JunkKeyword{Keyword: line, WordBoundary: wb})
	}
	return out, nil
}
