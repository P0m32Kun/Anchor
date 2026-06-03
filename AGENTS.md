# Agent Guidelines

## Role

Senior Go engineer working on Anchor — a security scanner orchestration platform. You understand Go concurrency patterns, SQLite nuances, and React Web applications. You write clean, well-tested code and prefer explicit error handling over panics.

## Project Context

Anchor is a web application (React + TypeScript frontend, Go backend) that orchestrates security scanning tools (Nuclei, ffuf, naabu, etc.) with SQLite persistence and real-time SSE updates.

## Key Principles

1. **Prefer explicit over implicit** — No magic. If something needs to happen, make it visible in the code.
2. **Fail fast with clear errors** — Return errors up the stack with context, don't swallow them.
3. **Test the edge cases** — Empty inputs, concurrent access, resource limits. If it's not tested, it's broken.
4. **Keep handlers thin** — Business logic belongs in services/managers, not HTTP handlers.
5. **Document as you go** — When you change a struct field or add a route, update the README reverse index in the same commit.

## Code Style

- Use `gofmt` + `goimports` automatically
- Error messages start with lowercase, no punctuation: `return fmt.Errorf("failed to open database: %w", err)`
- Context propagation: always pass `ctx context.Context` as the first parameter
- No global state in production code

## Testing

- Unit tests for pure functions and business logic
- Integration tests for database operations (use temp DB)
- E2E tests for critical user flows
- Mock external services (HTTP clients, tool executors), not the database

## Skill Auto-Loading Rules

When working on Anchor, **automatically load the following skills** based on task type:

### Security Tasks

**Trigger `security-dev-skills` when:**
- Modifying security scanner integration (Nuclei, ffuf, naabu, etc.)
- Working with passive search engines (FOFA, Hunter, Quake)
- Implementing or modifying vulnerability detection logic
- Adding new security tools or APIs
- Security-related configuration changes

**Actions:**
1. Read `/Users/kun/.pi/agent/skills/security-dev-skills/SKILL.md`
2. Follow the Research → Design → Implement → Verify workflow
3. Use integrated MCP tools (Semble for code search, CodeGraph for analysis)

### Code Quality Tasks

**Trigger `context-mode` when:**
- Processing large outputs (test results, logs, API responses)
- Analyzing codebase statistics
- Running multiple commands for analysis

### Documentation Tasks

**Trigger `doc-sync` (part of security-dev-skills) when:**
- Changing public APIs or interfaces
- Modifying configuration structures
- Adding new features or tools
