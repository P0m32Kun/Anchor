# Anchor Frontend E2E Tests

End-to-end tests using `agent-browser` CLI.

## Prerequisites

```bash
npm i -g agent-browser
agent-browser install
```

## Running Tests

### 1. Start the dev server

```bash
cd frontend
npm run dev
# Server runs on http://localhost:1420
```

### 2. Run E2E verification

```bash
# Navigate to app
agent-browser navigate http://localhost:1420

# Load the specific test plan from e2e/tests/*.e2e.md
# Follow the steps documented in each test file
```

## Test Coverage

| Page | Test File | Status |
|------|-----------|--------|
| App (routing + navbar) | `App.e2e.md` | ⬜ |
| Dashboard | `DashboardPage.e2e.md` | ⬜ |
| Projects | `ProjectPage.e2e.md` | ⬜ |
| Targets | `TargetPage.e2e.md` | ⬜ |
| Assets | `AssetPage.e2e.md` | ⬜ |
| Runs | `RunsPage.e2e.md` | ⬜ |
| Findings | `FindingsPage.e2e.md` | ⬜ |
| Reports | `ReportsPage.e2e.md` | ⬜ |
| Workers | `WorkersPage.e2e.md` | ⬜ |
| Settings | `SettingsPage.e2e.md` | ⬜ |

## Adding New Tests

When adding a new page or feature:

1. Create `e2e/tests/<PageName>.e2e.md` following the template in `_template.e2e.md`
2. Include: navigation, interaction, form submissions, error states
3. Mark disabled buttons and explain why
4. Verify all `<Link>` targets resolve to real routes
