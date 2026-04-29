# LegacyRouteGuard E2E Test Plan

## Test Purpose
Verify that legacy top-level routes (/targets, /assets, /runs, /findings, /reports) redirect to the corresponding nested project routes when a project is selected, and redirect to /projects when no project is selected.

## Prerequisite Steps
1. Start the dev server: `npm run dev` (app should be at http://localhost:1420)
2. Ensure the app has at least one project in the store, or test both states (with and without currentProjectId)

## Test Cases

### TC1: /targets redirects to /projects/:id/targets when project is selected
- **Navigation**: `navigate http://localhost:1420/targets`
- **Expected**: URL changes to `/projects/{lastProjectId}/targets` (where lastProjectId is the current project in store)
- **Validation**: `expect url toMatch /projects\/.*\/targets/`

### TC2: /assets redirects to /projects/:id/assets when project is selected
- **Navigation**: `navigate http://localhost:1420/assets`
- **Expected**: URL changes to `/projects/{lastProjectId}/assets`
- **Validation**: `expect url toMatch /projects\/.*\/assets/`

### TC3: /runs redirects to /projects/:id/runs when project is selected
- **Navigation**: `navigate http://localhost:1420/runs`
- **Expected**: URL changes to `/projects/{lastProjectId}/runs`
- **Validation**: `expect url toMatch /projects\/.*\/runs/`

### TC4: /findings redirects to /projects/:id/findings when project is selected
- **Navigation**: `navigate http://localhost:1420/findings`
- **Expected**: URL changes to `/projects/{lastProjectId}/findings`
- **Validation**: `expect url toMatch /projects\/.*\/findings/`

### TC5: /reports redirects to /projects/:id/reports when project is selected
- **Navigation**: `navigate http://localhost:1420/reports`
- **Expected**: URL changes to `/projects/{lastProjectId}/reports`
- **Validation**: `expect url toMatch /projects\/.*\/reports/`

### TC6: Legacy route redirects to /projects when no project is selected
- **Prerequisite**: Clear currentProjectId from localStorage (or visit on fresh profile)
- **Navigation**: `navigate http://localhost:1420/targets`
- **Expected**: URL changes to `/projects`
- **Validation**: `expect url toBe /projects`
