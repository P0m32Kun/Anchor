# Anchor

Anchor is a distributed internet attack-surface mapping platform for authorized adversary-emulation exercises. It turns targets into scoped assets and scanner work, coordinates worker execution, normalizes tool output into findings and evidence, tracks asset changes, streams progress, and supports human verification and report export.

The current repository is the first implementation base: it already contains an asset-driven engine, remote-worker paths, passive search integrations, and a React observation UI. The roadmap expands those seams into a reliable distributed mapping platform while keeping authorization, scope enforcement, and evidence quality central.

## Stack

- Go backend and scanner orchestration
- React 18 + TypeScript frontend
- SQLite in WAL mode (current implementation store; distributed storage evolution is planned)
- HTTP API with SSE updates
- Docker Compose deployment with server, worker, and frontend roles

## Quick start

Local development:

```bash
go run .
cd frontend && npm run dev
```

Production-style installation:

```bash
bash install.sh
```

Run repository documentation checks with `make check-docs`. Testing and deployment details live in the documentation hub.

## Documentation

[`docs/README.md`](docs/README.md) is the only documentation index. Start there for the product baseline, architecture, current plan, decisions, runbooks, references, and historical material.

Anchor may borrow ideas and interaction patterns from open-source projects, but third-party code or data is introduced only after license and provenance review. Capability research is a proposal until the current plan or an accepted decision adopts it.
