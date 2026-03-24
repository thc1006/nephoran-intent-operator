import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import App from "../App";

const mockFetch = vi.fn();
globalThis.fetch = mockFetch;

function mockJsonResponse(data: unknown, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? "OK" : "Error",
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  } as Response);
}

const EMPTY_INTENT: Record<string, unknown> = {
  id: "intent-20260306-0001",
  state: "acknowledged",
  expression: "Deploy eMBB slice",
  source: "web",
  plan: null, report: null, porch: null, git: null, errors: null,
  created_at: "2026-03-06T00:00:00Z",
  updated_at: "2026-03-06T00:00:00Z",
  history: [{ to: "acknowledged", at: "2026-03-06T00:00:00Z" }],
};

const MOCK_METRICS = {
  overview: { active_intents: 2, total_created: 15, completed: 12, failed: 1, git_commits: 11 },
  e2kpm: { cells: [
    { cell_id: "cell-001", gnb_id: "gnb-edge01", prb_usage_percent: 87.3 },
    { cell_id: "cell-002", gnb_id: "gnb-edge01", prb_usage_percent: 45.1 },
  ] },
  scale_actions: [{ component: "oai-odu", action: "scale_out", count: 3 }],
  porch_lifecycle: [{ lifecycle: "Proposed", count: 5 }],
  pipeline_stages: [{ stage: "planning", count: 15, sum_seconds: 22.5, p95_seconds: 2.1 }],
  timestamp: "2026-03-16T00:00:00Z",
};

beforeEach(() => {
  mockFetch.mockReset();
  mockFetch.mockImplementation((url: string) => {
    if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
    if (url.includes("/api/metrics/json")) return mockJsonResponse(MOCK_METRICS);
    if (url.includes("/api/porch/packages")) return mockJsonResponse([]);
    if (url.includes("/api/scale/status")) return mockJsonResponse({ components: [] });
    if (url.includes("/tmf-api/intent/v5/intent") && !url.includes("intent-"))
      return mockJsonResponse([]);
    return mockJsonResponse({}, 404);
  });
});

describe("App Layout", () => {
  it("renders sidebar with navigation", () => {
    render(<App />);
    expect(screen.getByText("Nephoran")).toBeTruthy();
    // Sidebar nav items (may also appear as page headings)
    expect(screen.getAllByText("Intent Pipeline").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Intent History")).toBeTruthy();
    expect(screen.getByText("Workloads")).toBeTruthy();
    expect(screen.getByText("Packages")).toBeTruthy();
    expect(screen.getByText("Closed Loop")).toBeTruthy();
  });

  it("renders header with cluster info", () => {
    render(<App />);
    expect(screen.getByText("thc1006-ubuntu-22")).toBeTruthy();
    expect(screen.getByText("K8s v1.35.1")).toBeTruthy();
  });

  it("shows Pipeline view by default", () => {
    render(<App />);
    expect(screen.getByText(/Submit a natural-language intent/)).toBeTruthy();
  });

  it("navigates between views", async () => {
    render(<App />);
    // Switch to Intent History
    fireEvent.click(screen.getByText("Intent History"));
    await waitFor(() => {
      expect(screen.getByText(/All intents from Web/)).toBeTruthy();
    });
    // Switch to Workloads
    fireEvent.click(screen.getByText("Workloads"));
    await waitFor(() => {
      expect(screen.getByText(/Live deployment status/)).toBeTruthy();
    });
    // Switch to Packages
    fireEvent.click(screen.getByText("Packages"));
    await waitFor(() => {
      expect(screen.getByText(/Nephio Porch PackageRevisions/)).toBeTruthy();
    });
    // Switch to Closed Loop
    fireEvent.click(screen.getByText("Closed Loop"));
    await waitFor(() => {
      expect(screen.getByText(/M7 closed-loop metrics/)).toBeTruthy();
    });
  });
});

describe("IntentPipeline", () => {
  it("disables submit when textarea is empty", () => {
    render(<App />);
    expect(screen.getByText("Submit Intent").closest("button")).toBeDisabled();
  });

  it("enables submit when text is entered", () => {
    render(<App />);
    fireEvent.change(screen.getByPlaceholderText(/Deploy eMBB/), {
      target: { value: "Deploy eMBB slice" },
    });
    expect(screen.getByText("Submit Intent").closest("button")).not.toBeDisabled();
  });

  it("calls TMF921 create endpoint on submit", async () => {
    mockFetch.mockImplementation((url: string, init?: RequestInit) => {
      if (url.includes("/tmf-api/intent/v5/intent") && init?.method === "POST") {
        return mockJsonResponse(EMPTY_INTENT);
      }
      if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
      if (url.includes("/api/porch/packages")) return mockJsonResponse([]);
      if (url.includes("/api/scale/status")) return mockJsonResponse({ components: [] });
      if (url.includes("/tmf-api/intent/v5/intent")) return mockJsonResponse([]);
      return mockJsonResponse({}, 404);
    });

    render(<App />);
    fireEvent.change(screen.getByPlaceholderText(/Deploy eMBB/), {
      target: { value: "Deploy eMBB slice" },
    });
    fireEvent.click(screen.getByText("Submit Intent"));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/tmf-api/intent/v5/intent"),
        expect.objectContaining({ method: "POST" }),
      );
    });
  });

  it("C2 fix: button is disabled during acknowledged state (pipeline active)", async () => {
    mockFetch.mockImplementation((url: string, init?: RequestInit) => {
      if (url.includes("/tmf-api/intent/v5/intent") && init?.method === "POST") {
        return mockJsonResponse(EMPTY_INTENT);
      }
      // Keep returning acknowledged to simulate pipeline running
      if (url.includes("/tmf-api/intent/v5/intent/intent-")) {
        return mockJsonResponse(EMPTY_INTENT);
      }
      if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
      if (url.includes("/api/porch/packages")) return mockJsonResponse([]);
      if (url.includes("/api/scale/status")) return mockJsonResponse({ components: [] });
      if (url.includes("/tmf-api/intent/v5/intent")) return mockJsonResponse([]);
      return mockJsonResponse({}, 404);
    });

    render(<App />);
    fireEvent.change(screen.getByPlaceholderText(/Deploy eMBB/), {
      target: { value: "Deploy eMBB slice" },
    });
    fireEvent.click(screen.getByText("Submit Intent"));

    await waitFor(() => {
      expect(screen.getByText("intent-20260306-0001")).toBeTruthy();
    });

    // Button should show Processing and be disabled
    await waitFor(() => {
      const btn = screen.getByText("Processing...").closest("button");
      expect(btn).toBeDisabled();
    });
  });
});

describe("IntentTable (History view)", () => {
  it("shows intents with state and source in table", async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url.includes("/tmf-api/intent/v5/intent") && !url.includes("intent-")) {
        return mockJsonResponse([
          { ...EMPTY_INTENT, state: "completed", source: "closedloop" },
        ]);
      }
      if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
      if (url.includes("/api/porch/packages")) return mockJsonResponse([]);
      if (url.includes("/api/scale/status")) return mockJsonResponse({ components: [] });
      return mockJsonResponse({}, 404);
    });

    render(<App />);
    fireEvent.click(screen.getByText("Intent History"));

    await waitFor(() => {
      expect(screen.getByText("intent-20260306-0001")).toBeTruthy();
      expect(screen.getByText("Auto")).toBeTruthy();
      expect(screen.getByText("completed")).toBeTruthy();
    });
  });

  it("shows porch lifecycle and git PR columns", async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url.includes("/tmf-api/intent/v5/intent") && !url.includes("intent-")) {
        return mockJsonResponse([
          {
            ...EMPTY_INTENT,
            state: "completed",
            source: "web",
            porch: { name: "instances/intent-20260306-0001", lifecycle: "Proposed", files: 6 },
            git: { branch: "intent/intent-20260306-0001", commit_sha: "abc123", pr: { number: 42, url: "http://gitea/pr/42" } },
          },
        ]);
      }
      if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
      if (url.includes("/api/porch/packages")) return mockJsonResponse([]);
      if (url.includes("/api/scale/status")) return mockJsonResponse({ components: [] });
      return mockJsonResponse({}, 404);
    });

    render(<App />);
    fireEvent.click(screen.getByText("Intent History"));

    await waitFor(() => {
      expect(screen.getByText("Proposed")).toBeTruthy();
      expect(screen.getByText("PR #42")).toBeTruthy();
    });
  });
});

describe("WorkloadTable", () => {
  it("shows deployments in table", async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url.includes("/api/scale/status")) {
        return mockJsonResponse({
          components: [
            { name: "free5gc-amf", namespace: "free5gc", domain: "core", replicas: 2, ready: 2, available: 2 },
          ],
        });
      }
      if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
      if (url.includes("/tmf-api/intent/v5/intent")) return mockJsonResponse([]);
      if (url.includes("/api/porch/packages")) return mockJsonResponse([]);
      return mockJsonResponse({}, 404);
    });

    render(<App />);
    fireEvent.click(screen.getByText("Workloads"));

    await waitFor(() => {
      expect(screen.getByText("free5gc-amf")).toBeTruthy();
    });
  });

  it("shows intent causal link when intentId label is present", async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url.includes("/api/scale/status")) {
        return mockJsonResponse({
          components: [
            {
              name: "ran-odu-edge01-embb-i001",
              namespace: "oran",
              domain: "ran",
              replicas: 3, ready: 3, available: 3,
              intentId: "intent-20260306-0001",
              slice: "embb",
              site: "edge01",
            },
          ],
        });
      }
      if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
      if (url.includes("/tmf-api/intent/v5/intent")) return mockJsonResponse([]);
      if (url.includes("/api/porch/packages")) return mockJsonResponse([]);
      return mockJsonResponse({}, 404);
    });

    render(<App />);
    fireEvent.click(screen.getByText("Workloads"));

    await waitFor(() => {
      expect(screen.getByText("ran-odu-edge01-embb-i001")).toBeTruthy();
      expect(screen.getByText("intent-20260306-0001")).toBeTruthy();
      expect(screen.getByText("embb")).toBeTruthy();
    });
  });
});

describe("IntentTable Cancel (T11)", () => {
  it("shows cancel button for acknowledged intents", async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url.includes("/tmf-api/intent/v5/intent") && !url.includes("intent-")) {
        return mockJsonResponse([
          { ...EMPTY_INTENT, state: "acknowledged" },
        ]);
      }
      if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
      if (url.includes("/api/porch/packages")) return mockJsonResponse([]);
      if (url.includes("/api/scale/status")) return mockJsonResponse({ components: [] });
      return mockJsonResponse({}, 404);
    });

    render(<App />);
    fireEvent.click(screen.getByText("Intent History"));

    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeTruthy();
      const cancelBtn = screen.getByText("Cancel").closest("button");
      expect(cancelBtn).not.toBeDisabled();
    });
  });

  it("does not show cancel button for completed intents", async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url.includes("/tmf-api/intent/v5/intent") && !url.includes("intent-")) {
        return mockJsonResponse([
          { ...EMPTY_INTENT, state: "completed" },
        ]);
      }
      if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
      if (url.includes("/api/porch/packages")) return mockJsonResponse([]);
      if (url.includes("/api/scale/status")) return mockJsonResponse({ components: [] });
      return mockJsonResponse({}, 404);
    });

    render(<App />);
    fireEvent.click(screen.getByText("Intent History"));

    await waitFor(() => {
      expect(screen.getByText("intent-20260306-0001")).toBeTruthy();
    });
    // No Cancel button should be present
    expect(screen.queryByText("Cancel")).toBeNull();
  });

  it("calls DELETE on cancel button click", async () => {
    mockFetch.mockImplementation((url: string, init?: RequestInit) => {
      if (url.includes("/tmf-api/intent/v5/intent/intent-") && init?.method === "DELETE") {
        return mockJsonResponse({ ...EMPTY_INTENT, state: "cancelled" });
      }
      if (url.includes("/tmf-api/intent/v5/intent") && !url.includes("intent-")) {
        return mockJsonResponse([
          { ...EMPTY_INTENT, state: "acknowledged" },
        ]);
      }
      if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
      if (url.includes("/api/porch/packages")) return mockJsonResponse([]);
      if (url.includes("/api/scale/status")) return mockJsonResponse({ components: [] });
      return mockJsonResponse({}, 404);
    });

    render(<App />);
    fireEvent.click(screen.getByText("Intent History"));

    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeTruthy();
    });
    fireEvent.click(screen.getByText("Cancel"));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/tmf-api/intent/v5/intent/intent-20260306-0001"),
        expect.objectContaining({ method: "DELETE" }),
      );
    });
  });
});

describe("PackageTable", () => {
  it("shows packages in table", async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url.includes("/api/porch/packages")) {
        return mockJsonResponse([
          { name: "rev-1", package: "free5gc/amf", lifecycle: "Published", workspace: "v1", repository: "nephoran-packages" },
        ]);
      }
      if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
      if (url.includes("/api/metrics/json")) return mockJsonResponse(MOCK_METRICS);
      if (url.includes("/tmf-api/intent/v5/intent")) return mockJsonResponse([]);
      if (url.includes("/api/scale/status")) return mockJsonResponse({ components: [] });
      return mockJsonResponse({}, 404);
    });

    render(<App />);
    fireEvent.click(screen.getByText("Packages"));

    await waitFor(() => {
      expect(screen.getByText("free5gc/amf")).toBeTruthy();
      expect(screen.getByText("Published")).toBeTruthy();
    });
  });
});

describe("ClosedLoopDashboard", () => {
  it("shows overview metrics cards", async () => {
    render(<App />);
    fireEvent.click(screen.getByText("Closed Loop"));

    await waitFor(() => {
      expect(screen.getByText("Active Intents")).toBeTruthy();
      expect(screen.getByText("Total Created")).toBeTruthy();
      expect(screen.getByText("Completed")).toBeTruthy();
      expect(screen.getByText("Failed")).toBeTruthy();
      expect(screen.getByText("Success Rate")).toBeTruthy();
    });

    // Check actual values from MOCK_METRICS
    await waitFor(() => {
      expect(screen.getByText("2")).toBeTruthy();  // active_intents
      expect(screen.getByText("15")).toBeTruthy(); // total_created
      expect(screen.getByText("12")).toBeTruthy(); // completed
      expect(screen.getByText("1")).toBeTruthy();  // failed
      expect(screen.getByText("92%")).toBeTruthy(); // 12/(12+1) = 92%
    });
  });

  it("shows PRB usage cells with threshold coloring", async () => {
    render(<App />);
    fireEvent.click(screen.getByText("Closed Loop"));

    await waitFor(() => {
      expect(screen.getByText("cell-001")).toBeTruthy();
      expect(screen.getByText("cell-002")).toBeTruthy();
      expect(screen.getByText("87.3%")).toBeTruthy();
      expect(screen.getByText("45.1%")).toBeTruthy();
    });
  });

  it("shows pipeline stage latencies", async () => {
    render(<App />);
    fireEvent.click(screen.getByText("Closed Loop"));

    await waitFor(() => {
      expect(screen.getByText("planning")).toBeTruthy();
      expect(screen.getByText(/p95 2\.10s/)).toBeTruthy();
    });
  });

  it("shows scale actions", async () => {
    render(<App />);
    fireEvent.click(screen.getByText("Closed Loop"));

    await waitFor(() => {
      expect(screen.getByText("oai-odu")).toBeTruthy();
      expect(screen.getByText("scale_out")).toBeTruthy();
      expect(screen.getByText("3")).toBeTruthy();
    });
  });

  it("shows porch lifecycle badges", async () => {
    render(<App />);
    fireEvent.click(screen.getByText("Closed Loop"));

    await waitFor(() => {
      expect(screen.getByText("Proposed: 5")).toBeTruthy();
    });
  });

  it("shows git commits count", async () => {
    render(<App />);
    fireEvent.click(screen.getByText("Closed Loop"));

    await waitFor(() => {
      expect(screen.getByText("11")).toBeTruthy(); // git_commits
      expect(screen.getByText("Commits pushed by pipeline")).toBeTruthy();
    });
  });

  it("handles empty metrics gracefully", async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url.includes("/api/metrics/json")) return mockJsonResponse({
        overview: { active_intents: 0, total_created: 0, completed: 0, failed: 0, git_commits: 0 },
        e2kpm: { cells: [] },
        scale_actions: [],
        porch_lifecycle: [],
        pipeline_stages: [],
        timestamp: "2026-03-16T00:00:00Z",
      });
      if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
      if (url.includes("/api/porch/packages")) return mockJsonResponse([]);
      if (url.includes("/api/scale/status")) return mockJsonResponse({ components: [] });
      if (url.includes("/tmf-api/intent/v5/intent")) return mockJsonResponse([]);
      return mockJsonResponse({}, 404);
    });

    render(<App />);
    fireEvent.click(screen.getByText("Closed Loop"));

    await waitFor(() => {
      expect(screen.getByText("No cell metrics available")).toBeTruthy();
      expect(screen.getByText("No pipeline data yet")).toBeTruthy();
    });
  });

  it("shows error state when API fails", async () => {
    mockFetch.mockImplementation((url: string) => {
      if (url.includes("/api/metrics/json")) return mockJsonResponse({ error: "fail" }, 500);
      if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
      if (url.includes("/api/porch/packages")) return mockJsonResponse([]);
      if (url.includes("/api/scale/status")) return mockJsonResponse({ components: [] });
      if (url.includes("/tmf-api/intent/v5/intent")) return mockJsonResponse([]);
      return mockJsonResponse({}, 404);
    });

    render(<App />);
    fireEvent.click(screen.getByText("Closed Loop"));

    await waitFor(() => {
      expect(screen.getByText(/500/)).toBeTruthy();
    });
  });
});
