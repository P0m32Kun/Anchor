package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// ParseGauOutput parses gau plain-text output (one URL per line).
func ParseGauOutput(r io.Reader) ([]string, error) {
	var urls []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			urls = append(urls, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return urls, fmt.Errorf("gau scan: %w", err)
	}
	return urls, nil
}

// ParseGauOutputBytes is a convenience wrapper for byte slices.
func ParseGauOutputBytes(data []byte) ([]string, error) {
	return ParseGauOutput(bytes.NewReader(data))
}
