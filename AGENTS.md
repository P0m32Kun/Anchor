<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **Anchor** (5740 symbols, 9586 relationships, 286 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/Anchor/context` | Codebase overview, check index freshness |
| `gitnexus://repo/Anchor/clusters` | All functional areas |
| `gitnexus://repo/Anchor/processes` | All execution flows |
| `gitnexus://repo/Anchor/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->

## Documentation Source Priority

Before using repository docs as truth, read them in this order:

1. `README.md`
2. `docs/README.md`
3. `docs/current/agent-guide.md`
4. `docs/current/plan.md`
5. `docs/current/architecture.md`
6. `docs/current/design/README.md` only if the task is explicitly about a proposal

Default assumptions:

- `docs/current/plan.md` is the only repository-wide plan.
- `docs/current/architecture.md` is the only repository-wide architecture baseline.
- documents under `docs/design/` are proposals unless promoted.
- documents under `docs/archived/`, `docs/active/review/`, and old redirect files are historical context, not current truth.

Do not treat the following as current truth unless the user explicitly asks for them:

- `plan.md`
- `docs/v0.3-plan.md`
- `docs/active/plan/`
- `docs/active/review/`
- `docs/refactoring-plan.md`
- `frontend/DESIGN_REFACTOR_PLAN.md`
