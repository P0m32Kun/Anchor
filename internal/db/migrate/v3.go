package db

import (
	"database/sql"
	"fmt"
)

// migrateV02 handles v0.2 schema migrations.
func migrateV02(db *sql.DB) error {
	// Add steps_json and tool_template_id to scan_tasks
	cols := []struct {
		table string
		name  string
		def   string
	}{
		{"projects", "port_range", "TEXT"},
		{"scan_tasks", "steps_json", "TEXT"},
		{"scan_tasks", "tool_template_id", "TEXT"},
		{"findings", "raw_request", "TEXT"},
		{"findings", "raw_response", "TEXT"},
		{"findings", "matched_template", "TEXT"},
	}
	for _, col := range cols {
		var exists bool
		err := db.QueryRow(
			`SELECT 1 FROM pragma_table_info(?) WHERE name = ?`,
			col.table, col.name,
		).Scan(&exists)
		if err == nil && exists {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, col.table, col.name, col.def)); err != nil {
			return fmt.Errorf("add column %s to %s: %w", col.name, col.table, err)
		}
	}

	// Insert default tool templates if none exist
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tool_templates`).Scan(&count); err != nil {
		return fmt.Errorf("count tool_templates: %w", err)
	}
	if count == 0 {
		templates := []struct {
			id, name, desc, profile, tools, severity string
			concurrency                              int
			screenshot                               bool
		}{
			{
				id: "external-quick", name: "外网快速摸底", desc: "轻量端口扫描 + 存活探测 + 高危漏洞",
				profile: "external", tools: `[{"tool":"naabu","enabled":true,"rate":1000},{"tool":"httpx","enabled":true,"rate":50},{"tool":"nuclei","enabled":true,"rate":50}]`,
				severity: "critical,high", concurrency: 10, screenshot: false,
			},
			{
				id: "external-standard", name: "外网标准初筛", desc: "完整资产发现 + 标准漏洞扫描",
				profile: "external", tools: `[{"tool":"subfinder","enabled":true,"rate":100},{"tool":"naabu","enabled":true,"rate":1000},{"tool":"httpx","enabled":true,"rate":50},{"tool":"nuclei","enabled":true,"rate":50}]`,
				severity: "critical,high,medium", concurrency: 20, screenshot: true,
			},
			{
				id: "internal-slow", name: "内网慢扫", desc: "低并发内网巡检，禁用高影响模板",
				profile: "internal", tools: `[{"tool":"naabu","enabled":true,"rate":100},{"tool":"httpx","enabled":true,"rate":10},{"tool":"nuclei","enabled":true,"rate":10}]`,
				severity: "critical,high", concurrency: 5, screenshot: false,
			},
			{
				id: "retest", name: "复测模式", desc: "只扫描已确认 Finding 对应目标",
				profile: "external", tools: `[{"tool":"nuclei","enabled":true,"rate":10}]`,
				severity: "critical,high,medium", concurrency: 5, screenshot: false,
			},
		}
		for _, t := range templates {
			_, err := db.Exec(`
				INSERT INTO tool_templates (id, name, description, profile_type, tools_json, default_max_concurrency, screenshot_enabled, nuclei_severity_filter)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, t.id, t.name, t.desc, t.profile, t.tools, t.concurrency, t.screenshot, t.severity)
			if err != nil {
				return fmt.Errorf("insert template %s: %w", t.id, err)
			}
		}
	}
	if err := migrateV02RunID(db); err != nil {
		return fmt.Errorf("migrate v0.2 run_id: %w", err)
	}
	return nil
}

func migrateV02RunID(db *sql.DB) error {
	// Check if column exists
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('scan_tasks') WHERE name = 'run_id'`).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = db.Exec(`ALTER TABLE scan_tasks ADD COLUMN run_id TEXT REFERENCES runs(id) ON DELETE CASCADE`)
		return err
	}
	return nil
}
