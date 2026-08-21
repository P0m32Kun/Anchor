# Deployment runbook

- Status: operational
- Updated: 2026-08-20

Use this runbook for packaged deployment. Use [`e2e-testing.md`](e2e-testing.md) for test stacks.

## Install

```bash
bash install.sh
```

The installer and Compose files are the source of truth for required variables, image names, ports, volumes, and health checks. Review their diff before a release; this runbook does not duplicate those values.

## Supported shapes

- `docker-compose.yml`: combined packaged deployment.
- `docker-compose.server.yml`: server/frontend side.
- `docker-compose.worker.yml`: worker side.

Use `make up`, `make down`, `make up-server`, or `make up-worker` for their defined lifecycle. `make` targets and `docker compose config` show the exact commands and resolved configuration.

## Safety checks

- Generate and store the API token as a secret; do not place it in source control, shell history, screenshots, or logs.
- Confirm data and work directories are persistent and writable before scanning.
- Confirm worker reachability and tool health from the UI/API before a long run.
- Back up the SQLite database and associated artifact directories together before upgrade.
- Use only authorized targets and review exclusions and rate limits before execution.
- Keep the E2E and release-verification Compose projects separate from user data.

## Upgrade and rollback

1. Record the running revision/images and take a consistent backup.
2. Review migrations and configuration changes.
3. Pull the intended images and inspect `docker compose config` without exposing secrets.
4. Start services and observe health, authentication, worker registration, and a bounded smoke flow.
5. Roll back images and restore the matched backup if a migration or smoke check fails.

Release automation and evidence are described in [`ci-cd.md`](ci-cd.md).
