import { describe, it, expect, vi } from "vitest";

describe("API client helpers", () => {
  it("buildQueryString omits undefined/empty params", () => {
    const { buildQueryString } = await import("./api");
    expect(buildQueryString("/projects", { page: 1, page_size: 10, search: undefined }))
      .toBe("/projects?page=1&page_size=10");
  });

  it("APIError marks NETWORK_ERROR and HTTP_5xx as retryable", () => {
    const { APIError } = await import("./api");
    const networkErr = new APIError("net", "NETWORK_ERROR");
    expect(networkErr.retryable).toBe(true);

    const fourErr = new APIError("4xx", "HTTP_4xx");
    expect(fourErr.retryable).toBe(false);
  });
});
