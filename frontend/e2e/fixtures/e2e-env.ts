/** Shared E2E defaults — keep in sync with Makefile `test-e2e*` targets. */
export const E2E_API_BASE =
	process.env.ANCHOR_API_BASE || "http://localhost:17421";

export const E2E_API_TOKEN =
	process.env.ANCHOR_API_TOKEN || "test-e2e-token";

/** When set (e.g. by `make test-e2e-smoke`), Playwright global setup skips Docker. */
export const E2E_SKIP_DOCKER = process.env.ANCHOR_E2E_SKIP_DOCKER === "1";
