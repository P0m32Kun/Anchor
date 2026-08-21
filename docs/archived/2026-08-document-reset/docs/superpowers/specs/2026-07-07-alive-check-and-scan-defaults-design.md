# Alive check and scan defaults alignment design

Date: 2026-07-07

## Scope

This change set does three things, in order:

1. Add a real IP liveness gate for all scan flows.
2. Make ScanModal prefer deployment defaults from `GET /scan/defaults`.
3. Sync stale comments and docs to the implemented behavior.

It does not add new UI controls for alive-scan tuning in this pass.

## Problem

Current behavior has three mismatches:

- The product describes `nmap alive -> Naabu -> nmap -sV`, but there is no explicit alive-check action.
- Any IP with unknown `Alive` state can still proceed to later work.
- ScanModal initializes from frontend constants, so deployment YAML defaults can drift from what users see.

## Goals

- Every IP must be probed for liveness before later scans.
- Dead IPs must not produce port scan or later work.
- The alive-check must be visible as a first-class action/stage.
- ScanModal should show deployment defaults when available.
- Docs and comments should describe the real behavior.

## Non-goals

- No new per-mode alive-check strategy knobs.
- No separate ICMP-vs-ARP UI.
- No attempt to optimize batching beyond the current worker pattern unless needed by the existing code path.

## Chosen approach

Add a new scan action, `ALIVE_CHECK`, implemented with `nmap -sn`, and make it the hard prerequisite for any IP-based follow-up work.

Why this approach:

- It matches the intended product behavior.
- It keeps the scheduler honest by exposing the step as a stage instead of burying it inside port scan.
- It gives one place to enforce the rule: no alive result, no later IP scanning.

## Behavior design

### 1. New action and stage

Add:

- `ActionAliveCheck`
- tool mapping to a dedicated tool id for nmap alive check
- stage mapping, e.g. `alive`

Stage order becomes:

- discovery
- subdomain
- resolve
- cdn
- alive
- port
- service
- web
- crawl
- brute
- vuln

### 2. Asset eligibility rules

For every `IP` and `CIDR` asset, derive `ALIVE_CHECK` first.

Rules after this change:

- `ALIVE_CHECK` is the only IP-based action allowed before liveness is known.
- `PORT_SCAN` requires `Alive == true`.
- Any direct IP-based `HTTPX` candidate path must also require `Alive == true`.
- Dead IPs stop there.
- Unknown liveness is no longer treated as good enough.

For `CIDR`, alive-check expands to host-level probing through the same execution path used by the command builder. The implementation should reuse the existing seed/asset model already used for IP targets instead of inventing a new asset family.

### 3. Command execution

Use `nmap -sn` as the first version.

Command shape:

- single IP: `nmap -sn <ip>`
- batch file or range form only if the existing worker/executor already has a clean way to do it without adding new framework code

The lazy rule here is: keep the first implementation aligned with the current worker command-builder style. Prefer one new command builder and one new parser over any new execution subsystem.

### 4. Result handling

Alive-check output updates the asset attrs:

- reachable host -> `Alive=true`
- unreachable host -> `Alive=false`

Only an alive result may cause later derived work for that IP.

### 5. ScanModal defaults

ScanModal should initialize defaults in this order:

1. deployment defaults from `GET /scan/defaults`
2. saved project config from `GET /projects/{id}/pipeline/config`
3. localStorage/user overrides
4. frontend constants as fallback only

This keeps the UI aligned with deployment YAML while preserving per-project and per-user state.

### 6. Docs and comments

Sync these to the implemented behavior:

- remove or rewrite any text that implies alive-check is only descriptive
- document `alive -> portscan -> fingerprint -> httpx -> optional crawl/brute/spoor -> nuclei`
- note that all IP-based scanning is blocked on liveness

## Data flow

1. Seed target becomes `IP` or `CIDR` asset.
2. Engine derives `ALIVE_CHECK`.
3. Alive-check runs `nmap -sn`.
4. Parser marks host alive/dead.
5. Only alive IPs derive or release later work.
6. Port scan and later stages proceed as before.

## Testing

Minimal required tests:

1. command-builder unit test for the alive-check command
2. rules/profile test proving dead or unknown IPs do not derive later work
3. parser/result test proving alive/dead output updates attrs correctly
4. ScanModal test proving `/scan/defaults` values are preferred when present

Keep tests narrow; no new broad test harness.

## Risks

- `CIDR` handling can get messy if implemented as a special executor path. Avoid that unless the current worker model forces it.
- If the current engine relies on `Alive=nil` for bootstrap behavior, the rules change will surface hidden assumptions. That is acceptable; the new gate is intentional.

## Rollout note

This is a behavior-tightening change. Some runs will scan fewer hosts after it lands because dead hosts will be filtered earlier. That is expected.
