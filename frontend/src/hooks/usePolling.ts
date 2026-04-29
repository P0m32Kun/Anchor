import { useCallback, useEffect, useRef, useState } from "react";

export interface UsePollingOptions {
  /** Polling interval in ms; default 5000 */
  interval?: number;
  /** Whether polling is enabled; default true */
  enabled?: boolean;
  /** Pause polling when page is hidden; default true */
  pauseOnHidden?: boolean;
}

export interface UsePollingReturn<T> {
  data: T | null;
  isLoading: boolean;
  error: Error | null;
  /** Manually trigger a poll */
  refetch: () => void;
}

const DEFAULT_INTERVAL = 5000;

/**
 * usePolling — interval-based data fetching with visibility-aware pause.
 */
export function usePolling<T>(
  pollFn: () => Promise<T>,
  options: UsePollingOptions = {}
): UsePollingReturn<T> {
  const { interval = DEFAULT_INTERVAL, enabled = true, pauseOnHidden = true } = options;

  const [data, setData] = useState<T | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollFnRef = useRef(pollFn);
  const enabledRef = useRef(enabled);
  const pauseOnHiddenRef = useRef(pauseOnHidden);

  useEffect(() => {
    pollFnRef.current = pollFn;
  }, [pollFn]);

  useEffect(() => {
    enabledRef.current = enabled;
  }, [enabled]);

  useEffect(() => {
    pauseOnHiddenRef.current = pauseOnHidden;
  }, [pauseOnHidden]);

  const executePoll = useCallback(async () => {
    if (!enabledRef.current) return;
    if (pauseOnHiddenRef.current && document.hidden) return;

    setIsLoading(true);
    try {
      const result = await pollFnRef.current();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setIsLoading(false);
    }
  }, []);

  const refetch = useCallback(() => {
    executePoll();
  }, [executePoll]);

  useEffect(() => {
    if (!enabled) return;

    // Initial poll
    executePoll();

    intervalRef.current = setInterval(() => {
      executePoll();
    }, interval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [enabled, interval, executePoll]);

  // Visibility change: when page becomes visible, trigger an immediate poll
  useEffect(() => {
    const handleVisibility = () => {
      if (!document.hidden && enabledRef.current && pauseOnHiddenRef.current) {
        executePoll();
      }
    };
    document.addEventListener("visibilitychange", handleVisibility);
    return () => document.removeEventListener("visibilitychange", handleVisibility);
  }, [executePoll]);

  return { data, isLoading, error, refetch };
}
