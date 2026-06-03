/**
 * 日志审计审计通道(§3.4)
 *
 * 在 pipeline 运行结束后,通过 GET /tasks/{id}/output 拉取 stdout/stderr,
 * 用规则包断言 CLI 层行为。
 *
 * 规则包放在 fixtures/log-rules/ 下,每条规则包是一个 JSON 文件。
 *
 * ---
 *
 * API 响应格式:
 *   GET /tasks/{id}/output?stream=stdout&offset=0
 *   → { stream, offset, content, done }
 *
 * 规则类型:
 *   - mustContain: content 必须包含某字符串
 *   - notContain:  content 必须不含某字符串
 *   - regex:       content 必须匹配某正则
 *   - stderrEmpty: stderr 必须为空(或只含已知白名单模式)
 */

import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { E2E_API_BASE, E2E_API_TOKEN } from "./e2e-env";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const API_BASE = E2E_API_BASE;
const API_TOKEN = E2E_API_TOKEN;
const RULES_DIR = path.resolve(__dirname, "log-rules");

// --- Types ---

export interface LogRule {
	type: "mustContain" | "notContain" | "regex" | "stderrEmpty";
	value?: string;
	scope: "stdout" | "stderr";
}

export interface LogRulePackage {
	tool: string;
	rules: LogRule[];
}

export interface AuditResult {
	tool: string;
	passed: boolean;
	failures: string[];
}

// --- Task output fetching ---

interface TaskOutputResponse {
	stream: string;
	offset: number;
	content: string;
	done: boolean;
}

/**
 * Fetch the full stdout/stderr for a completed task.
 * Iterates with offset until done=true.
 */
async function fetchTaskOutput(
	taskId: string,
	stream: "stdout" | "stderr",
): Promise<string> {
	let offset = 0;
	let fullContent = "";

	for (let i = 0; i < 100; i++) {
		const url = `${API_BASE}/tasks/${taskId}/output?stream=${stream}&offset=${offset}`;
		const res = await fetch(url, {
			headers: { Authorization: `Bearer ${API_TOKEN}` },
		});
		if (!res.ok) {
			throw new Error(
				`GET /tasks/${taskId}/output failed: ${res.status} ${res.statusText}`,
			);
		}
		const body: TaskOutputResponse = await res.json();
		fullContent += body.content;
		offset = body.offset;
		if (body.done) break;
	}

	return fullContent;
}

// --- Rule evaluation ---

function evaluateRule(content: string, rule: LogRule): string | null {
	switch (rule.type) {
		case "mustContain":
			if (!content.includes(rule.value!)) {
				return `stdout 缺少 "${rule.value}"`;
			}
			return null;

		case "notContain":
			if (content.includes(rule.value!)) {
				return `stdout 包含不应出现的 "${rule.value}"`;
			}
			return null;

		case "regex":
			try {
				const re = new RegExp(rule.value!);
				if (!re.test(content)) {
					return `stdout 不匹配正则为 "${rule.value}"`;
				}
			} catch (e) {
				return `正则 "${rule.value}" 语法错误: ${e}`;
			}
			return null;

		case "stderrEmpty": {
			const stripped = content
				.split("\n")
				.map((l) => l.trim())
				.filter(Boolean);
			// 已知白名单模式: 不含 error/warn/panic 等关键词的 stderr 行可忽略
			const knownPatterns = [
				/^go: downloading /,
				/^go: found /,
				/^go: added /,
				/^time="?2\d{3}/,
				/^level=(info|debug|warn)/i,
				/^\d{4}\/\d{2}\/\d{2}/,
			];
			const unexpected = stripped.filter(
				(line) => !knownPatterns.some((p) => p.test(line)),
			);
			if (unexpected.length > 0) {
				return `stderr 存在疑似错误: ${unexpected.slice(0, 5).join("; ")}`;
			}
			return null;
		}

		default:
			return `未知规则类型: ${(rule as any).type}`;
	}
}

// --- Main audit function ---

/**
 * Run log audit for a list of task IDs against their tool-specific rule packages.
 *
 * Usage in test:
 *   const results = await auditTaskLogs(tasks, ["naabu"]);
 *   for (const r of results) {
 *     expect(r.passed).toBe(true);
 *   }
 */
export async function auditTaskLogs(
	tasks: Array<{ id: string; source_tool?: string }>,
	toolNames: string[],
): Promise<AuditResult[]> {
	const results: AuditResult[] = [];

	for (const toolName of toolNames) {
		// Load rule package
		const rulePath = path.join(RULES_DIR, `${toolName}.json`);
		if (!fs.existsSync(rulePath)) {
			results.push({
				tool: toolName,
				passed: false,
				failures: [`未找到规则包: ${rulePath}`],
			});
			continue;
		}
		const pkg: LogRulePackage = JSON.parse(
			fs.readFileSync(rulePath, "utf-8"),
		);

		// Find matching tasks
		const matchingTasks = tasks.filter(
			(t) => t.source_tool === toolName,
		);
		if (matchingTasks.length === 0) {
			results.push({
				tool: toolName,
				passed: false,
				failures: [`没有找到 source_tool="${toolName}" 的任务`],
			});
			continue;
		}

		// Evaluate rules on each task
		const failures: string[] = [];
		for (const task of matchingTasks) {
			const stdoutTask = await fetchTaskOutput(task.id, "stdout");
			const stderrTask = await fetchTaskOutput(task.id, "stderr");

			for (const rule of pkg.rules) {
				const content =
					rule.scope === "stdout" ? stdoutTask : stderrTask;
				const fail = evaluateRule(content, rule);
				if (fail) {
					failures.push(`[task ${task.id}] ${fail}`);
				}
			}
		}

		results.push({
			tool: toolName,
			passed: failures.length === 0,
			failures,
		});
	}

	return results;
}

/**
 * Simplified audit: fetch a single task's stdout/stderr and check inline rules.
 * For tests that need ad-hoc log assertions beyond rule packages.
 */
export async function singleTaskAudit(
	taskId: string,
	rules: LogRule[],
): Promise<AuditResult> {
	const stdout = await fetchTaskOutput(taskId, "stdout");
	const stderr = await fetchTaskOutput(taskId, "stderr");
	const failures: string[] = [];

	for (const rule of rules) {
		const content = rule.scope === "stdout" ? stdout : stderr;
		const fail = evaluateRule(content, rule);
		if (fail) {
			failures.push(fail);
		}
	}

	return {
		tool: taskId,
		passed: failures.length === 0,
		failures,
	};
}

// --- Helpers for test code ---

/**
 * Print audit results in a readable format. Call in afterAll or on failure.
 */
export function printAuditResults(results: AuditResult[]): void {
	for (const r of results) {
		const icon = r.passed ? "✓" : "✗";
		console.log(`[log-audit] ${icon} ${r.tool}: ${r.passed ? "通过" : "失败"}`);
		if (r.failures.length > 0) {
			for (const f of r.failures) {
				console.log(`  └─ ${f}`);
			}
		}
	}
}
