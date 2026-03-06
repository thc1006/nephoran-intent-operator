/** API client for intentd v2 — TMF921 async API. */

const BASE = import.meta.env.VITE_API_BASE ?? "";

async function json<T>(url: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {};
  if (init?.body) headers["Content-Type"] = "application/json";

  const res = await fetch(`${BASE}${url}`, { headers, ...init });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`${res.status} ${res.statusText}: ${body}`);
  }
  return res.json();
}

// ---- Types ----------------------------------------------------------------

export interface IntentRecord {
  id: string;
  state: string;
  expression: string;
  source: string;
  plan: Record<string, unknown> | null;
  report: Record<string, unknown> | null;
  porch: { name: string; lifecycle: string; files?: number; error?: string } | null;
  git: { branch: string; commit_sha: string; pr?: { number: number; url: string } } | null;
  errors: string[] | null;
  created_at: string;
  updated_at: string;
  history: Array<{
    from?: string;
    to: string;
    at: string;
    data?: Record<string, unknown> | null;
  }>;
}

export interface PorchPackage {
  name: string;
  package: string;
  lifecycle: string;
  workspace: string;
  repository: string;
}

export interface ScaleComponent {
  name: string;
  namespace: string;
  domain: string;
  replicas: number;
  ready: number;
  available: number;
  intentId?: string | null;
  slice?: string | null;
  site?: string | null;
}

export interface ScaleStatus {
  components: ScaleComponent[];
  timestamp?: string;
  error?: string;
}

// ---- TMF921 API -----------------------------------------------------------

export function createIntent(
  expression: string,
  source = "web",
  useLlm = true,
  dryRun = false,
): Promise<IntentRecord> {
  return json<IntentRecord>("/tmf-api/intent/v5/intent", {
    method: "POST",
    body: JSON.stringify({ expression, source, use_llm: useLlm, dry_run: dryRun }),
  });
}

export function getIntent(id: string): Promise<IntentRecord> {
  return json<IntentRecord>(`/tmf-api/intent/v5/intent/${id}`);
}

export function listIntents(): Promise<IntentRecord[]> {
  return json<IntentRecord[]>("/tmf-api/intent/v5/intent");
}

export function deleteIntent(id: string): Promise<IntentRecord> {
  return json<IntentRecord>(`/tmf-api/intent/v5/intent/${id}`, { method: "DELETE" });
}

export interface IntentReport {
  intentId: string;
  state: string;
  expression: string;
  source: string;
  compliant: boolean;
  terminal: boolean;
  stageCount: number;
  createdAt: string;
  updatedAt: string;
  plan: Record<string, unknown> | null;
  porch: Record<string, unknown> | null;
  git: Record<string, unknown> | null;
  errors: string[] | null;
  report: Record<string, unknown> | null;
  history: IntentRecord["history"];
}

export function getIntentReport(id: string): Promise<IntentReport> {
  return json<IntentReport>(`/tmf-api/intent/v5/intent/${id}/report`);
}

// ---- Helper endpoints -----------------------------------------------------

export function listPorchPackages(): Promise<PorchPackage[]> {
  return json<PorchPackage[]>("/api/porch/packages");
}

export function getScaleStatus(): Promise<ScaleStatus> {
  return json<ScaleStatus>("/api/scale/status");
}

export function healthz(): Promise<{ status: string }> {
  return json<{ status: string }>("/healthz");
}

// ---- State helpers --------------------------------------------------------

export const PIPELINE_STATES = [
  "acknowledged",
  "planning",
  "validating",
  "generating",
  "executing",
  "proposed",
  "applied",
  "completed",
] as const;

export type PipelineState = (typeof PIPELINE_STATES)[number];

export function stateIndex(state: string): number {
  return PIPELINE_STATES.indexOf(state as PipelineState);
}

export function isTerminal(state: string): boolean {
  return state === "completed" || state === "failed" || state === "cancelled";
}

/** C2 fix: acknowledged is considered active for UI — pipeline has started. */
export function isActive(state: string): boolean {
  return !isTerminal(state);
}
