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

beforeEach(() => {
  mockFetch.mockReset();
  mockFetch.mockImplementation((url: string) => {
    if (url.includes("/healthz")) return mockJsonResponse({ status: "ok" });
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
