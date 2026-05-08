package parser

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"strconv"
	"strings"
)

// ParseNaabu reads Naabu output (JSONL or CSV) and returns discovered hosts/ports/ips.
func ParseNaabu(r io.Reader) ([]NaabuResult, []ParseError) {
	// Peek at first non-empty line to determine format.
	buf := bufio.NewReader(r)
	var firstLine string
	for {
		line, err := buf.ReadString('\n')
		if err != nil && line == "" {
			return nil, nil
		}
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			firstLine = trimmed
			break
		}
	}

	// Reconstruct reader with the peeked line prepended.
	remaining := buf
	preread := strings.NewReader(firstLine + "\n")
	combined := io.MultiReader(preread, remaining)

	if strings.HasPrefix(firstLine, "{") {
		return parseNaabuJSONL(combined)
	}
	return parseNaabuCSV(combined)
}

func parseNaabuJSONL(r io.Reader) ([]NaabuResult, []ParseError) {
	return parseJSONLines(r, func(line []byte, lineNo int) (NaabuResult, ParseError) {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			return NaabuResult{}, ParseError{Line: lineNo, Raw: string(line), Message: "invalid JSON: " + err.Error()}
		}

		res := NaabuResult{}

		if b, ok := raw["host"]; ok {
			_ = json.Unmarshal(b, &res.Host)
		}
		if b, ok := raw["ip"]; ok {
			_ = json.Unmarshal(b, &res.IP)
		}
		if b, ok := raw["port"]; ok {
			var portInt int
			if err := json.Unmarshal(b, &portInt); err == nil {
				res.Port = portInt
			} else {
				var portStr string
				if err := json.Unmarshal(b, &portStr); err == nil {
					if p, err := strconv.Atoi(portStr); err == nil {
						res.Port = p
					}
				}
			}
		}

		if res.Host == "" && res.IP == "" {
			return res, ParseError{Line: lineNo, Raw: string(line), Message: "missing host and ip"}
		}
		if res.Port == 0 {
			return res, ParseError{Line: lineNo, Raw: string(line), Message: "missing or zero port"}
		}

		return res, ParseError{}
	})
}

func parseNaabuCSV(r io.Reader) ([]NaabuResult, []ParseError) {
	var results []NaabuResult
	var errs []ParseError

	reader := csv.NewReader(r)
	// Naabu CSV has no header by default; columns: host,ip,port
	lineNo := 0
	for {
		lineNo++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errs = append(errs, ParseError{Line: lineNo, Raw: strings.Join(record, ","), Message: "csv read error: " + err.Error()})
			continue
		}
		if len(record) < 2 {
			errs = append(errs, ParseError{Line: lineNo, Raw: strings.Join(record, ","), Message: "too few columns"})
			continue
		}

		res := NaabuResult{}
		if len(record) >= 1 {
			res.Host = strings.TrimSpace(record[0])
		}
		if len(record) >= 2 {
			res.IP = strings.TrimSpace(record[1])
		}
		if len(record) >= 3 {
			if p, err := strconv.Atoi(strings.TrimSpace(record[2])); err == nil {
				res.Port = p
			}
		}

		if res.Host == "" && res.IP == "" {
			errs = append(errs, ParseError{Line: lineNo, Raw: strings.Join(record, ","), Message: "missing host and ip"})
			continue
		}
		if res.Port == 0 {
			errs = append(errs, ParseError{Line: lineNo, Raw: strings.Join(record, ","), Message: "missing or zero port"})
			continue
		}

		results = append(results, res)
	}

	return results, errs
}

// ParseNaabuOutput parses naabu -json JSONL output into port info.
func ParseNaabuOutput(r io.Reader) []PortInfo {
	results, _ := ParseNaabu(r)
	var ports []PortInfo
	for _, res := range results {
		if res.Port > 0 {
			ports = append(ports, PortInfo{
				IP:       res.IP,
				Port:     res.Port,
				Protocol: "tcp",
			})
		}
	}
	return ports
}
