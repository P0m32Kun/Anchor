# Frontend convention

- Status: normative
- Updated: 2026-08-20

The current frontend is React 18, TypeScript, Vite, Tailwind, Zustand, and the API client under `frontend/src/lib/`. Package manifests and source code define exact versions and structure.

## Rules

- Route-level pages compose focused components; shared behavior belongs in hooks or `lib` rather than duplicated effects.
- Use the existing API client and authentication flow instead of ad hoc `fetch` calls in components.
- Model loading, empty, error, and success states explicitly.
- Preserve server field names and enum values at the client boundary; map them deliberately when UI names differ.
- Clean up timers, subscriptions, and SSE connections on unmount or dependency change.
- Keep destructive actions explicit and confirm them where recovery is difficult.
- Maintain keyboard access, labels, focus behavior, and readable status feedback.
- Add focused unit coverage for changed component behavior and E2E coverage for critical journeys.

Do not copy another project's component implementation or visual assets without license review. A UI reference may influence information architecture while Anchor keeps its own React implementation.

Verification follows [`testing.md`](testing.md); full-stack commands are in [`../runbooks/e2e-testing.md`](../runbooks/e2e-testing.md).
