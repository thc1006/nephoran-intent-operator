import { useEffect, useRef, useState } from "react";

/** Poll a fetcher function at a fixed interval. */
export function usePolling<T>(
  fetcher: () => Promise<T>,
  intervalMs: number,
  enabled = true,
): { data: T | null; error: string | null; loading: boolean } {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    if (!enabled) return;

    let timer: ReturnType<typeof setInterval>;

    async function tick() {
      try {
        const result = await fetcher();
        if (mountedRef.current) {
          setData(result);
          setError(null);
        }
      } catch (e) {
        if (mountedRef.current) setError(String(e));
      } finally {
        if (mountedRef.current) setLoading(false);
      }
    }

    tick();
    timer = setInterval(tick, intervalMs);

    return () => {
      mountedRef.current = false;
      clearInterval(timer);
    };
  }, [fetcher, intervalMs, enabled]);

  return { data, error, loading };
}
