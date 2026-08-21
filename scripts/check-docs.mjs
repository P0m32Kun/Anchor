import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const REQUIRED = [
  "AGENTS.md",
  "README.md",
  "CHANGELOG.md",
  "docs/README.md",
  "docs/current/product.md",
  "docs/current/architecture.md",
  "docs/current/plan.md",
  "docs/decisions/README.md",
  "docs/decisions/0001-document-authority.md",
  "docs/decisions/0002-product-baseline.md",
  "docs/decisions/0003-distributed-internet-asm-baseline.md",
];

const BASELINE_MARKERS = new Map([
  ["AGENTS.md", "distributed internet attack-surface mapping platform"],
  ["README.md", "distributed internet attack-surface mapping platform"],
  ["docs/current/product.md", "Dedicated internal-network scanning is not a product scenario."],
  ["docs/current/architecture.md", "## Legacy internal mode"],
  ["docs/current/plan.md", "Do not reintroduce a dedicated internal-network product scenario."],
]);

const CURRENT = new Set([
  "docs/current/product.md",
  "docs/current/architecture.md",
  "docs/current/plan.md",
]);

const ALLOWED_DOC_SECTIONS = new Set([
  "README.md",
  "functional-test.md",
  "archived",
  "conventions",
  "current",
  "decisions",
  "pitfalls",
  "proposals",
  "reference",
  "runbooks",
  "templates",
]);

const SKIP_DIRECTORIES = new Set([
  ".agent",
  ".claude",
  ".codex",
  ".deepseek",
  ".git",
  ".gstack",
  ".worktrees",
  "node_modules",
  "dist",
  "playwright-report",
  "tasks",
  "test-results",
  "tmp-test",
]);

const RETIRED_PATTERNS = [
  [/docs\/current\/(?:agent-guide|deployment|e2e-testing|ci-cd-guide|faq|scan-api-guide|decisions|design)/, "retired docs/current path"],
  [/docs\/(?:active|superpowers|mempalace|features|plans)\//, "retired live documentation section"],
  [/internal\/workflow(?:\/|\b)/, "deleted internal/workflow path"],
  [/~\/(?:\.p-skills|\.pi)\//, "host-specific skill path"],
  [/\/Users\/[A-Za-z0-9._-]+\//, "personal absolute path"],
  [/\b(?:develop-feature|security-dev-skills|context-mode|doc-sync)\b/, "unavailable skill dependency"],
];

const STATUS_RULES = [
  [/^docs\/current\//, /- Status: current\b/],
  [/^docs\/conventions\//, /- Status: normative\b/],
  [/^docs\/proposals\//, /- Status: proposal\b/],
  [/^docs\/reference\//, /- Status: reference\b/],
  [/^docs\/runbooks\//, /- Status: operational\b/],
  [/^docs\/decisions\/\d{4}-/, /- Status: (?:accepted|superseded by \d{4})\b/],
];

function walk(root, directory = root, files = []) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    if (entry.isSymbolicLink()) continue;
    const absolute = path.join(directory, entry.name);
    const relative = path.relative(root, absolute).replaceAll(path.sep, "/");
    if (entry.isDirectory()) {
      if (SKIP_DIRECTORIES.has(entry.name) || relative === "docs/archived") continue;
      walk(root, absolute, files);
    } else if (entry.isFile() && entry.name.endsWith(".md")) {
      files.push(relative);
    }
  }
  return files;
}

function markdownTargets(source) {
  const targets = [];
  const pattern = /!?\[[^\]]*\]\(([^)]+)\)/g;
  for (const match of source.matchAll(pattern)) {
    let target = match[1].trim();
    if (target.startsWith("<") && target.endsWith(">")) target = target.slice(1, -1);
    const titleStart = target.search(/\s+["']/);
    if (titleStart !== -1) target = target.slice(0, titleStart);
    targets.push(target);
  }
  return targets;
}

function isExternal(target) {
  return /^(?:https?:|mailto:|data:|app:)/i.test(target) || target.startsWith("#");
}

export function checkDocs(rootPath) {
  const root = fs.realpathSync(rootPath);
  const errors = [];
  const fail = (file, message) => errors.push({ file, message });

  for (const required of REQUIRED) {
    if (!fs.existsSync(path.join(root, required))) fail(required, "required document is missing");
  }

  for (const [file, marker] of BASELINE_MARKERS) {
    const absolute = path.join(root, file);
    if (fs.existsSync(absolute) && !fs.readFileSync(absolute, "utf8").includes(marker)) {
      fail(file, `missing internet-only product baseline marker: ${marker}`);
    }
  }

  const currentDirectory = path.join(root, "docs/current");
  if (fs.existsSync(currentDirectory)) {
    const actual = walk(root, currentDirectory).filter((file) => file.endsWith(".md"));
    for (const file of actual) {
      if (!CURRENT.has(file)) fail(file, "only product.md, architecture.md, and plan.md may be current");
    }
    for (const file of CURRENT) {
      if (!actual.includes(file)) fail(file, "canonical current document is missing");
    }
  }

  const docsDirectory = path.join(root, "docs");
  if (fs.existsSync(docsDirectory)) {
    for (const entry of fs.readdirSync(docsDirectory, { withFileTypes: true })) {
      if (!ALLOWED_DOC_SECTIONS.has(entry.name)) {
        fail(`docs/${entry.name}`, "unrecognized live documentation section");
      }
    }
  }

  const files = walk(root).sort();
  for (const file of files) {
    const absolute = path.join(root, file);
    const source = fs.readFileSync(absolute, "utf8");

    if (/source_of_truth:\s*true/i.test(source)) {
      fail(file, "source_of_truth declarations are retired; authority is defined in docs/README.md");
    }

    for (const [scope, requiredStatus] of STATUS_RULES) {
      if (scope.test(file) && !requiredStatus.test(source)) {
        fail(file, `missing legal status declaration ${requiredStatus}`);
      }
    }

    const enforceRetiredReferences = !file.startsWith("docs/pitfalls/");
    if (enforceRetiredReferences) {
      for (const [pattern, description] of RETIRED_PATTERNS) {
        if (pattern.test(source)) fail(file, description);
      }
    }

    for (const rawTarget of markdownTargets(source)) {
      if (!rawTarget || isExternal(rawTarget)) continue;
      if (rawTarget.startsWith("/")) {
        fail(file, `absolute local link is not portable: ${rawTarget}`);
        continue;
      }
      let target;
      try {
        target = decodeURIComponent(rawTarget.split("#", 1)[0].split("?", 1)[0]);
      } catch {
        fail(file, `invalid percent-encoding in link: ${rawTarget}`);
        continue;
      }
      if (!target) continue;
      const resolved = path.resolve(path.dirname(absolute), target);
      const relative = path.relative(root, resolved);
      if (relative === ".." || relative.startsWith(`..${path.sep}`)) {
        fail(file, `local link escapes repository: ${rawTarget}`);
      } else if (!fs.existsSync(resolved)) {
        fail(file, `broken local link: ${rawTarget}`);
      }
    }
  }

  return errors;
}

function runCli() {
  const root = process.argv[2] ? path.resolve(process.argv[2]) : process.cwd();
  const errors = checkDocs(root);
  if (errors.length > 0) {
    process.stderr.write(`Documentation check failed with ${errors.length} issue(s):\n`);
    for (const error of errors) process.stderr.write(`- ${error.file}: ${error.message}\n`);
    process.exitCode = 1;
    return;
  }
  process.stdout.write("Documentation check passed.\n");
}

if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) runCli();
