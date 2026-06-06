import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../lib/api";

export type SSEStatus = "connecting" | "open" | "closed" | "error";

export interface UseSSEOptions {
  onMessage?: (data: unknown) => void;
  onError?: (err: Error) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  reconnect?: boolean;
  maxRetries?: number;
  /** Heartbeat timeout in ms; default 30000 */
  heartbeatTimeout?: number;
  /** Project ID to filter events and fetch SSE token */
  projectId?: string;
}

export interface UseSSEReturn {
  status: SSEStatus;
  retryCount: number;
  /** Manually trigger a reconnection */
  reconnect: () => void;
  /** Manually close the connection */
  close: () => void;
}

const DEFAULT_HEARTBEAT_TIMEOUT = 30000;
const DEFAULT_MAX_RETRIES = 5;
const INITIAL_RETRY_DELAY = 1000;
const MAX_RETRY_DELAY = 30000;

/**
 * Fetch a short-lived SSE token for the given project.
 * Returns the token string, or null if the fetch fails.
 */
async function fetchSSEToken(projectId: string): Promise<string | null> {
  try {
    const data = await api.fetchSSEToken(projectId);
    return data.token;
  } catch {
    return null;
  }
}

/**
 * useSSE — robust EventSource hook with auto-reconnect, heartbeat detection,
 * page-visibility management, and max-retry cap.
 *
 * When projectId is provided, automatically fetches a short-lived SSE JWT token
 * and passes it via query parameter, avoiding the EventSource header limitation.
 */
export function useSSE(url: string, options: UseSSEOptions = {}): UseSSEReturn {
  const {
    onMessage,
    onError,
    onConnect,
    onDisconnect,
    reconnect: enableReconnect = true,
    maxRetries = DEFAULT_MAX_RETRIES,
    heartbeatTimeout = DEFAULT_HEARTBEAT_TIMEOUT,
    projectId,
  } = options;

  const [status, setStatus] = useState<SSEStatus>("closed");
  const [retryCount, setRetryCount] = useState(0);

  const esRef = useRef<EventSource | null>(null);
  const timersRef = useRef({
    reconnect: null as ReturnType<typeof setTimeout> | null,
    heartbeat: null as ReturnType<typeof setTimeout> | null,
  });
  const retryDelayRef = useRef(INITIAL_RETRY_DELAY);
  const retryCountRef = useRef(0);
  const isManualCloseRef = useRef(false);
  const wasOpenRef = useRef(false);

  // Keep callbacks in refs so we don't re-bind listeners on every render
  const callbacksRef = useRef({ onMessage, onError, onConnect, onDisconnect });
  useEffect(() => {
    callbacksRef.current = { onMessage, onError, onConnect, onDisconnect };
  }, [onMessage, onError, onConnect, onDisconnect]);

  // Use refs to break circular dependency between connect ↔ scheduleReconnect
  const connectRef = useRef<(() => void) | undefined>(undefined);
  const scheduleReconnectRef = useRef<(() => void) | undefined>(undefined);

  const clearTimers = useCallback(() => {
    Object.values(timersRef.current).forEach((id) => {
      if (id) clearTimeout(id);
    });
    timersRef.current = { reconnect: null, heartbeat: null };
  }, []);

  const closeConnection = useCallback(() => {
    clearTimers();
    if (esRef.current) {
      esRef.current.close();
      esRef.current = null;
    }
  }, [clearTimers]);

  const scheduleReconnect = useCallback(() => {
    if (timersRef.current.reconnect) return;

    timersRef.current.reconnect = setTimeout(() => {
      timersRef.current.reconnect = null;
      connectRef.current?.();
    }, retryDelayRef.current);

    retryDelayRef.current = Math.min(retryDelayRef.current * 2, MAX_RETRY_DELAY);
  }, []);

  const resetHeartbeat = useCallback(() => {
    if (timersRef.current.heartbeat) clearTimeout(timersRef.current.heartbeat);

    timersRef.current.heartbeat = setTimeout(() => {
      closeConnection();
      setStatus("closed");
      callbacksRef.current.onDisconnect?.();
      wasOpenRef.current = false;
      if (enableReconnect && retryCountRef.current < maxRetries) {
        scheduleReconnectRef.current?.();
      }
    }, heartbeatTimeout);
  }, [closeConnection, enableReconnect, maxRetries, heartbeatTimeout]);

  const connect = useCallback(async () => {
    if (esRef.current) return; // already connected or connecting

    isManualCloseRef.current = false;
    setStatus("connecting");

    // Build the URL with project_id filter
    let fullUrl = url;
    if (projectId) {
      const sep = url.includes("?") ? "&" : "?";
      fullUrl = `${url}${sep}project_id=${encodeURIComponent(projectId)}`;
    }

    // Fetch SSE token if projectId is available
    if (projectId) {
      const sseToken = await fetchSSEToken(projectId);
      if (sseToken) {
        const sep = fullUrl.includes("?") ? "&" : "?";
        fullUrl = `${fullUrl}${sep}token=${encodeURIComponent(sseToken)}`;
      }
      // If token fetch fails, try without token (will rely on Bearer auth if possible)
    }

    // Check if we were manually closed while waiting for the token
    if (isManualCloseRef.current) return;

    try {
      const es = new EventSource(fullUrl);
      esRef.current = es;

      es.onopen = () => {
        retryDelayRef.current = INITIAL_RETRY_DELAY;
        retryCountRef.current = 0;
        setRetryCount(0);
        setStatus("open");
        wasOpenRef.current = true;
        callbacksRef.current.onConnect?.();
        resetHeartbeat();
      };

      es.onmessage = (event) => {
        resetHeartbeat();

        let parsed: unknown;
        try {
          parsed = JSON.parse(event.data);
        } catch {
          parsed = event.data;
        }

        callbacksRef.current.onMessage?.(parsed);
      };

      es.onerror = () => {
        closeConnection();
        const nextRetry = retryCountRef.current + 1;
        retryCountRef.current = nextRetry;
        setRetryCount(nextRetry);

        if (nextRetry >= maxRetries) {
          setStatus("error");
          callbacksRef.current.onError?.(
            new Error(`SSE connection failed after ${maxRetries} retries`)
          );
          wasOpenRef.current = false;
          return;
        }

        setStatus("closed");
        callbacksRef.current.onDisconnect?.();
        wasOpenRef.current = false;

        if (enableReconnect && !isManualCloseRef.current) {
          scheduleReconnectRef.current?.();
        }
      };
    } catch (err) {
      setStatus("error");
      callbacksRef.current.onError?.(
        err instanceof Error ? err : new Error(String(err))
      );
    }
  }, [url, projectId, maxRetries, enableReconnect, closeConnection, resetHeartbeat]);

  // Bind refs to latest function instances (breaks circular dependency)
  connectRef.current = connect;
  scheduleReconnectRef.current = scheduleReconnect;

  const reconnect = useCallback(() => {
    isManualCloseRef.current = false;
    retryCountRef.current = 0;
    setRetryCount(0);
    retryDelayRef.current = INITIAL_RETRY_DELAY;
    closeConnection();
    connect();
  }, [closeConnection, connect]);

  const close = useCallback(() => {
    isManualCloseRef.current = true;
    wasOpenRef.current = false;
    closeConnection();
    setStatus("closed");
    setRetryCount(0);
    retryCountRef.current = 0;
    retryDelayRef.current = INITIAL_RETRY_DELAY;
  }, [closeConnection]);

  // Initial connect + cleanup
  useEffect(() => {
    connect();
    return () => {
      isManualCloseRef.current = true;
      closeConnection();
    };
  }, [connect, closeConnection]);

  // Page visibility management
  useEffect(() => {
    const handleVisibility = () => {
      if (document.hidden) {
        // Page hidden: close connection to save resources
        if (esRef.current) {
          closeConnection();
          setStatus("closed");
          callbacksRef.current.onDisconnect?.();
        }
      } else {
        // Page visible: reconnect if we were previously open
        if (wasOpenRef.current && !esRef.current && !isManualCloseRef.current) {
          reconnect();
        }
      }
    };

    document.addEventListener("visibilitychange", handleVisibility);
    return () => document.removeEventListener("visibilitychange", handleVisibility);
  }, [closeConnection, reconnect]);

  return { status, retryCount, reconnect, close };
}
