import { useState } from "react";
import Sidebar from "./components/Sidebar";
import Header from "./components/Header";
import IntentPipeline from "./components/IntentPipeline";
import IntentTable from "./components/IntentTable";
import WorkloadTable from "./components/WorkloadTable";
import PackageTable from "./components/PackageTable";
import ClosedLoopDashboard from "./components/ClosedLoopDashboard";

export type View = "pipeline" | "intents" | "workloads" | "packages" | "closedloop";

export default function App() {
  const [view, setView] = useState<View>("pipeline");

  return (
    <div className="min-h-screen flex bg-gray-950">
      <Sidebar current={view} onNavigate={setView} />

      <div className="flex-1 flex flex-col min-w-0">
        <Header />

        <main className="flex-1 overflow-auto p-6">
          {view === "pipeline" && <IntentPipeline />}
          {view === "intents" && <IntentTable />}
          {view === "workloads" && <WorkloadTable />}
          {view === "packages" && <PackageTable />}
          {view === "closedloop" && <ClosedLoopDashboard />}
        </main>
      </div>
    </div>
  );
}
