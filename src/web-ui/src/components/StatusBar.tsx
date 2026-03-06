import { useCallback, useEffect, useState } from "react";
import { healthz } from "../api/client";

export default function StatusBar() {
  const [status, setStatus] = useState<"ok" | "error" | "loading">("loading");

  const check = useCallback(async () => {
    try {
      const res = await healthz();
      setStatus(res.status === "ok" ? "ok" : "error");
    } catch {
      setStatus("error");
    }
  }, []);

  useEffect(() => {
    check();
    const timer = setInterval(check, 15000);
    return () => clearInterval(timer);
  }, [check]);

  return (
    <header className="flex items-center justify-between border-b border-gray-800 px-6 py-3">
      <div className="flex items-center gap-3">
        <h1 className="text-xl font-bold tracking-tight">Nephoran Intent Console</h1>
        <span className="text-xs text-gray-500">TMF921 + Porch + GitOps</span>
      </div>
      <div className="flex items-center gap-2 text-xs">
        <span className="text-gray-500">intentd</span>
        <span
          className={`inline-block h-2 w-2 rounded-full ${
            status === "ok"
              ? "bg-green-500"
              : status === "error"
                ? "bg-red-500"
                : "bg-gray-600 animate-pulse"
          }`}
        />
      </div>
    </header>
  );
}
