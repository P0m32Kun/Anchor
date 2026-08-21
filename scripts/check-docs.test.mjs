import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";

import { checkDocs } from "./check-docs.mjs";

function fixture(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "anchor-docs-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const write = (file, source) => {
    const absolute = path.join(root, file);
    fs.mkdirSync(path.dirname(absolute), { recursive: true });
    fs.writeFileSync(absolute, source);
  };

  write("AGENTS.md", "# Agent contract\n\ndistributed internet attack-surface mapping platform\n");
  write("CLAUDE.md", "@AGENTS.md\n");
  write("README.md", "distributed internet attack-surface mapping platform\n[Docs](docs/README.md)\n");
  write("CHANGELOG.md", "# Changelog\n");
  write("docs/README.md", "[Product](current/product.md)\n");
  write("docs/functional-test.md", "# Acceptance\n");
  write("docs/current/product.md", "# Product\n\n- Status: current\n\nDedicated internal-network scanning is not a product scenario.\n");
  write("docs/current/architecture.md", "# Architecture\n\n- Status: current\n\n## Legacy internal mode\n");
  write("docs/current/plan.md", "# Plan\n\n- Status: current\n\nDo not reintroduce a dedicated internal-network product scenario.\n");
  write("docs/decisions/README.md", "# Decisions\n");
  write("docs/decisions/0001-document-authority.md", "# ADR\n\n- Status: accepted\n");
  write("docs/decisions/0002-product-baseline.md", "# ADR\n\n- Status: superseded by 0003\n");
  write("docs/decisions/0003-distributed-internet-asm-baseline.md", "# ADR\n\n- Status: accepted\n");
  return { root, write };
}

test("accepts one coherent authority chain", (t) => {
  const { root } = fixture(t);
  assert.deepEqual(checkDocs(root), []);
});

test("rejects a fourth current document", (t) => {
  const { root, write } = fixture(t);
  write("docs/current/roadmap.md", "# Competing roadmap\n\n- Status: current\n");
  assert.ok(checkDocs(root).some((issue) => issue.message.includes("only product.md")));
});

test("rejects duplicate authority declarations and retired dependencies", (t) => {
  const { root, write } = fixture(t);
  write("docs/reference/trap.md", "# Trap\n\n- Status: reference\nsource_of_truth: true\nUse develop-feature.\n");
  const messages = checkDocs(root).map((issue) => issue.message);
  assert.ok(messages.some((message) => message.includes("source_of_truth")));
  assert.ok(messages.some((message) => message.includes("unavailable skill")));
});

test("rejects broken links but ignores archived snapshots", (t) => {
  const { root, write } = fixture(t);
  write("docs/reference/live.md", "# Live\n\n- Status: reference\n[Missing](missing.md)\n");
  write("docs/archived/old.md", "[Also missing](missing.md)\nsource_of_truth: true\n");
  const issues = checkDocs(root);
  assert.equal(issues.filter((issue) => issue.message.includes("broken local link")).length, 1);
  assert.ok(issues.every((issue) => !issue.file.startsWith("docs/archived/")));
});

test("rejects missing lifecycle status and a duplicated Claude contract", (t) => {
  const { root, write } = fixture(t);
  write("docs/proposals/idea.md", "# Idea without status\n");
  write("CLAUDE.md", "# Separate rules\n");
  const issues = checkDocs(root);
  assert.ok(issues.some((issue) => issue.message.includes("missing legal status")));
  assert.ok(issues.some((issue) => issue.file === "CLAUDE.md" && issue.message.includes("thin pointer")));
});

test("rejects loss of the internet-only product boundary", (t) => {
  const { root, write } = fixture(t);
  write("docs/current/product.md", "# Product\n\n- Status: current\n\nInternal and external scanning are both product modes.\n");
  assert.ok(checkDocs(root).some((issue) => issue.message.includes("internet-only product baseline marker")));
});
