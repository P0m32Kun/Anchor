import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../lib/api";
import type { ScanTask } from "../lib/api";

const POLL_MS = 2000;

function isLiveTaskStatus(status: string): boolean {
  return status === "running" || status === "queued";
}

function formatLiveOutput(stdout: string, stderr: string): string {
  const parts: string[] = [];
  if (stdout) parts.push(stdout);
  if (stderr) {
    if (parts.length > 0) parts.push("\n--- stderr ---\n");
    parts.push(stderr);
  }
  return parts.join("") || "";
}

/**
 * Polls GET /tasks/{id}/output while a task is running so raw logs appear before completion.
 */
export function useTaskLiveOutput(task: ScanTask | null, enabled: boolean) {
  const [text, setText] = useState("");
  const [loading, setLoading] = useState(false);
  const stdoutRef = useRef("");
  const stderrRef = useRef("");
  const offsetRef = useRef({ stdout: 0, stderr: 0 });

  const reset = useCallback(() => {
    stdoutRef.current = "";
    stderrRef.current = "";
    offsetRef.current = { stdout: 0, stderr: 0 };
    setText("");
  }, []);

  useEffect(() => {
    if (!enabled || !task || !isLiveTaskStatus(task.status)) {
      if (!enabled) reset();
      return;
    }

    let cancelled = false;
    setLoading(true);

    const poll = async () => {
      try {
        const [out, errOut] = await Promise.all([
          api.getTaskOutput(task.id, { stream: "stdout", offset: offsetRef.current.stdout }),
          api.getTaskOutput(task.id, { stream: "stderr", offset: offsetRef.current.stderr }),
        ]);
        if (cancelled) return;
        if (out.content) {
          stdoutRef.current += out.content;
          offsetRef.current.stdout = out.offset;
        }
        if (errOut.content) {
          stderrRef.current += errOut.content;
          offsetRef.current.stderr = errOut.offset;
        }
        setText(formatLiveOutput(stdoutRef.current, stderrRef.current));
      } catch {
        if (!cancelled) {
          setText((prev) => prev || "(拉取实时输出失败，稍后重试)");
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    reset();
    void poll();
    const id = window.setInterval(() => void poll(), POLL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [task?.id, task?.status, enabled, reset]);

  return { text, loading };
}
