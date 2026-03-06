import { useCallback, useEffect, useState } from "react";
import { Activity } from "lucide-react";
import { healthz } from "../api/client";

export default function Header() {
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
    const t = setInterval(check, 15000);
    return () => clearInterval(t);
  }, [check]);

  return (
    <header className="h-14 flex items-center justify-between border-b border-gray-800 bg-gray-900/50 px-6">
      <div className="flex items-center gap-3 text-sm text-gray-400">
        <span className="text-gray-500">Cluster:</span>
        <span className="text-gray-200 font-medium">thc1006-ubuntu-22</span>
        <span className="text-gray-700">|</span>
        <span className="text-gray-500">K8s v1.35.1</span>
      </div>

      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2 text-xs">
          <Activity size={14} className="text-gray-500" />
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
          <span className={`text-xs ${status === "ok" ? "text-green-500" : status === "error" ? "text-red-400" : "text-gray-500"}`}>
            {status === "ok" ? "Healthy" : status === "error" ? "Down" : "Checking"}
          </span>
        </div>
      </div>
    </header>
  );
}
