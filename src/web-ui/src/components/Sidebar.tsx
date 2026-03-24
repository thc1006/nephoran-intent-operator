import { Zap, List, Server, Package, Activity } from "lucide-react";
import type { View } from "../App";

const NAV: Array<{ id: View; label: string; icon: typeof Zap }> = [
  { id: "pipeline", label: "Intent Pipeline", icon: Zap },
  { id: "intents", label: "Intent History", icon: List },
  { id: "workloads", label: "Workloads", icon: Server },
  { id: "packages", label: "Packages", icon: Package },
  { id: "closedloop", label: "Closed Loop", icon: Activity },
];

interface Props {
  current: View;
  onNavigate: (v: View) => void;
}

export default function Sidebar({ current, onNavigate }: Props) {
  return (
    <aside className="w-56 flex-shrink-0 border-r border-gray-800 bg-gray-900 flex flex-col">
      {/* Logo */}
      <div className="h-14 flex items-center px-5 border-b border-gray-800">
        <div className="flex items-center gap-2.5">
          <div className="w-7 h-7 rounded-md bg-blue-600 flex items-center justify-center">
            <span className="text-white text-xs font-bold">N</span>
          </div>
          <div>
            <div className="text-sm font-semibold text-gray-100 leading-tight">Nephoran</div>
            <div className="text-[10px] text-gray-500 leading-tight">Intent Operator</div>
          </div>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 py-3 px-3 space-y-0.5">
        {NAV.map(({ id, label, icon: Icon }) => {
          const active = current === id;
          return (
            <button
              key={id}
              onClick={() => onNavigate(id)}
              className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-md text-sm transition-colors ${
                active
                  ? "bg-blue-600/15 text-blue-400"
                  : "text-gray-400 hover:bg-gray-800 hover:text-gray-200"
              }`}
            >
              <Icon size={16} />
              {label}
            </button>
          );
        })}
      </nav>

      {/* Footer */}
      <div className="px-5 py-3 border-t border-gray-800 text-[10px] text-gray-600">
        TMF 921 / O-RAN WG6 / Porch v3
      </div>
    </aside>
  );
}
