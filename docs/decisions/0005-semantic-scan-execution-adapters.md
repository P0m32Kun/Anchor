# 0005. Use semantic scan requests with SDK and process adapters

- Status: accepted
- Date: 2026-08-20

## Context

The current execution path is centered on a registry that renders CLI arguments and a worker that executes the resulting command. ProjectDiscovery tools such as Naabu and Subfinder expose Go libraries with context-aware execution and callbacks, which can reduce deployment friction. Nuclei also exposes a Go SDK, but its documentation warns that the SDK is actively changing and that service-style use carries additional security risk. A single implementation choice would either bind the scan engine to CLI details or weaken process isolation for every tool.

## Decision

Introduce a semantic execution module whose interface accepts a typed `ScanRequest` and returns normalized results and an execution receipt. The module owns policy checks, adapter selection, option mapping, output parsing, cancellation, redaction, resource accounting, and provenance. It has two real adapters: an SDK adapter for tools that pass parity and safety evidence, and a guarded process adapter for tools that need isolation or lack a stable SDK.

Naabu is the first SDK pilot. Subfinder may follow after the same evidence gate. Nuclei remains an isolated process adapter initially; Nmap, FFUF, Katana, and Spoor remain process adapters until an equivalent SDK and isolation story exists. Remote workers receive semantic requests, never arbitrary command strings assembled by the control plane.

## Consequences

The scan engine and distributed protocol gain a stable, small seam and can evolve tool implementations locally. SDK adoption must account for raw-socket privileges, browser or native dependencies, memory and panic isolation, version churn, and transitive licenses; removing an external binary does not by itself prove easier or safer deployment. Every promoted adapter requires parity, cancellation, budget, provenance, resource, and recovery evidence.
