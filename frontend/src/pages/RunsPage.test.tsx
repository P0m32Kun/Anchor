import { describe, it, expect } from "vitest";
import { mergeStageEvent } from "./RunsPage";
import type { PipelineRunStage } from "../lib/api";

const mkStage = (over: Partial<PipelineRunStage>): PipelineRunStage => ({
  id: "id-1",
  run_id: "run-1",
  stage: "portscan",
  status: "running",
  created_at: "2026-05-12T00:00:00.000Z",
  ...over,
});

describe("mergeStageEvent (RunsPage SSE reducer)", () => {
  it("updates an existing stage's status without changing length", () => {
    const prev = [mkStage({ stage: "portscan", status: "running" })];
    const next = mergeStageEvent(prev, {
      run_id: "run-1",
      stage: "portscan",
      status: "completed",
    });
    expect(next).toHaveLength(1);
    expect(next[0].stage).toBe("portscan");
    expect(next[0].status).toBe("completed");
  });

  // REGRESSION: this is the heart of Fix 2 on the frontend. Slow-scan stages
  // (ffuf, urlfinder) arrive AFTER the initial
  // loadRunDetails snapshot was taken, so the reducer must append unknown
  // stages — not drop them. The pre-fix code returned `prev` unchanged when
  // findIndex < 0, which is exactly why slow-scan rows didn't show up in the UI.
  // We use a synthetic stage name here so future slow-scan tools added to the
  // pipeline also flow through this append path without needing test changes.
  it("appends a previously unseen stage (slow-scan path)", () => {
    const prev = [mkStage({ stage: "vuln", status: "completed" })];
    const next = mergeStageEvent(prev, {
      run_id: "run-1",
      stage: "future_slow_tool",
      status: "running",
    });
    expect(next).toHaveLength(2);
    expect(next[1].stage).toBe("future_slow_tool");
    expect(next[1].status).toBe("running");
    expect(next[1].id.startsWith("tmp-future_slow_tool-")).toBe(true);
    expect(next[1].started_at).toBeTruthy();
  });

  it("appends a failed stage with error string (Fix 3 backend backstop path)", () => {
    const prev = [mkStage({ stage: "vuln", status: "completed" })];
    const next = mergeStageEvent(prev, {
      run_id: "run-1",
      stage: "ffuf",
      status: "failed",
      error: "ffuf enabled but no dictionary configured",
    });
    expect(next).toHaveLength(2);
    expect(next[1].stage).toBe("ffuf");
    expect(next[1].status).toBe("failed");
    expect(next[1].error).toContain("no dictionary configured");
  });

  it("defaults status to 'running' when message omits it", () => {
    const prev: PipelineRunStage[] = [];
    const next = mergeStageEvent(prev, { run_id: "run-1", stage: "alive" });
    expect(next[0].status).toBe("running");
  });

  it("returns prev unchanged when msg.stage is missing", () => {
    const prev = [mkStage({ stage: "portscan" })];
    const next = mergeStageEvent(prev, { run_id: "run-1", status: "completed" });
    expect(next).toBe(prev);
  });
});
