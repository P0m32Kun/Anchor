import { describe, it, expect } from "vitest";
import { APIError, PAGE_ALL } from "./api";

describe("APIError", () => {
  it("marks NETWORK_ERROR as retryable", () => {
    const err = new APIError("net", "NETWORK_ERROR");
    expect(err.retryable).toBe(true);
  });

  it("marks HTTP_5xx as retryable", () => {
    const err = new APIError("5xx", "HTTP_5xx");
    expect(err.retryable).toBe(true);
  });

  it("marks HTTP_4xx as not retryable", () => {
    const err = new APIError("4xx", "HTTP_4xx");
    expect(err.retryable).toBe(false);
  });

  it("defaults retryable to false for UNKNOWN", () => {
    const err = new APIError("unknown");
    expect(err.retryable).toBe(false);
  });
});

describe("PAGE_ALL", () => {
  it("has a large page_size", () => {
    expect(PAGE_ALL.page_size).toBe(10000);
  });
});
