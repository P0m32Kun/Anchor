import { useState, useEffect, useCallback, useRef } from "react";

export interface ResourceState<T> {
  data: T;
  loading: boolean;
  error: string | null;
}

export interface UseResourceReturn<T> extends ResourceState<T> {
  reload: () => void;
}

/**
 * 通用数据加载 Hook，封装 AbortController、loading、error 状态管理。
 *
 * @param loader  返回 Promise<T> 的加载函数，支持 AbortSignal
 * @param deps    依赖数组，变化时自动重新加载
 * @param initialData  初始数据值
 */
export function useResource<T>(
  loader: (signal?: AbortSignal) => Promise<T>,
  deps: React.DependencyList,
  initialData: T
): UseResourceReturn<T> {
  const [data, setData] = useState<T>(initialData);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 用 ref 保存最新的 loader，避免 deps 变化时闭包捕获旧值
  const loaderRef = useRef(loader);
  loaderRef.current = loader;

  const load = useCallback(
    async (signal?: AbortSignal) => {
      setLoading(true);
      setError(null);
      try {
        const result = await loaderRef.current(signal);
        if (!signal?.aborted) {
          setData(result);
        }
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        if (!signal?.aborted) {
          setError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        if (!signal?.aborted) {
          setLoading(false);
        }
      }
    },
    [] // 内部使用 ref，不依赖外部 loader
  );

  const reload = useCallback(() => {
    const ctrl = new AbortController();
    load(ctrl.signal);
  }, [load]);

  useEffect(() => {
    const ctrl = new AbortController();
    load(ctrl.signal);
    return () => ctrl.abort();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return { data, loading, error, reload };
}
