import { useCallback, useEffect, useRef, useState } from "react";
import { useSSE, type UseSSEOptions } from "./useSSE";
import { usePolling } from "./usePolling";

export interface UseRealtimeDataOptions<T> {
  /** SSE endpoint URL (absolute or relative) */
  sseUrl: string;
  /** Polling function invoked when SSE is unavailable */
  pollFn: () => Promise<T>;
  /** Polling interval in ms; default 5000 */
  pollInterval?: number;
  /** SSE options passed through to useSSE */
  sseOptions?: UseSSEOptions;
  /** Callback whenever new data arrives (from SSE or polling) */
  onData?: (data: T) => void;
}

export interface UseRealtimeDataReturn<T> {
  /** Latest data (null until first successful fetch) */
  data: T | null;
  /** true when SSE is active; false when polling or unavailable */
  isLive: boolean;
  /** true while polling (not live) */
  isPolling: boolean;
  /** Combined loading state */
  isLoading: boolean;
  /** Last error, if any */
  error: Error | null;
  /** Force a refresh (reconnect SSE or trigger poll) */
  refresh: () => void;
}

/**
 * useRealtimeData — SSE-first with automatic polling fallback.
 *
 * When the SSE connection is healthy, data flows in real-time.
 * If SSE exhausts its retry budget, polling takes over transparently.
 * If the user calls refresh() or visibility returns, SSE is attempted again.
 */
export function useRealtimeData<T>(
  options: UseRealtimeDataOptions<T>
): UseRealtimeDataReturn<T> {
  const {
    sseUrl,
    pollFn,
    pollInterval = 5000,
    sseOptions = {},
    onData,
  } = options;

  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [sseFailed, setSseFailed] = useState(false);
  const [pollingEnabled, setPollingEnabled] = useState(false);

  const onDataRef = useRef(onData);
  useEffect(() => {
    onDataRef.current = onData;
  }, [onData]);

  // When SSE sends a message, wrap it as T and forward
  const handleSSEMessage = useCallback(
    (raw: unknown) => {
      // The SSE payload is already the data; cast it through
      const typed = raw as T;
      setData(typed);
      setError(null);
      onDataRef.current?.(typed);
    },
    []
  );

  // When SSE errors out (max retries exceeded), flip to polling
  const handleSSEError = useCallback((err: Error) => {
    setError(err);
    setSseFailed(true);
  }, []);

  const handleSSEConnect = useCallback(() => {
    setSseFailed(false);
    setError(null);
  }, []);

  const handleSSEDisconnect = useCallback(() => {
    // intentionally empty; disconnect alone doesn't mean failure
  }, []);

  const { status, reconnect: reconnectSSE } = useSSE(sseUrl, {
    ...sseOptions,
    onMessage: (raw) => {
      handleSSEMessage(raw);
      sseOptions?.onMessage?.(raw);
    },
    onError: (err) => {
      handleSSEError(err);
      sseOptions?.onError?.(err);
    },
    onConnect: () => {
      handleSSEConnect();
      sseOptions?.onConnect?.();
    },
    onDisconnect: () => {
      handleSSEDisconnect();
      sseOptions?.onDisconnect?.();
    },
  });

  // Delay polling startup by 3s when SSE fails to avoid jitter
  useEffect(() => {
    if (sseFailed && status === "error") {
      const timer = setTimeout(() => setPollingEnabled(true), 3000);
      return () => clearTimeout(timer);
    }
    setPollingEnabled(false);
  }, [sseFailed, status]);

  const pollingResult = usePolling<T>(pollFn, {
    interval: pollInterval,
    enabled: pollingEnabled,
    pauseOnHidden: true,
  });

  // Merge polling data into our state
  useEffect(() => {
    if (pollingEnabled && pollingResult.data !== null) {
      setData(pollingResult.data);
      if (pollingResult.error) {
        setError(pollingResult.error);
      } else {
        setError(null);
      }
    }
  }, [pollingEnabled, pollingResult.data, pollingResult.error]);

  const isLive = status === "open" && !sseFailed;
  const isPolling = pollingEnabled;
  const isLoading =
    status === "connecting" || (pollingEnabled && pollingResult.isLoading && data === null);

  const refresh = useCallback(() => {
    // Try SSE again first
    setSseFailed(false);
    setError(null);
    reconnectSSE();
  }, [reconnectSSE]);

  return {
    data,
    isLive,
    isPolling,
    isLoading,
    error,
    refresh,
  };
}
