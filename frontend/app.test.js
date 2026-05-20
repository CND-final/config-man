import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

const html = readFileSync(resolve(process.cwd(), "index.html"), "utf8");

let app;
let state;

const baseState = {
  token: "",
  user: null,
  activeView: "dashboard",
  activeProjectId: "",
  activeEnvironment: "prod",
  configSearch: "",
  diffFilter: "all",
  projects: [],
  configs: [],
  requests: [],
  templates: [],
  diffItems: [],
};

function loadHtml() {
  const parsed = new DOMParser().parseFromString(html, "text/html");
  document.documentElement.innerHTML = parsed.documentElement.innerHTML;
}

function resetState() {
  Object.assign(state, baseState);
  state.revealedKeys = new Set();
}

beforeAll(async () => {
  localStorage.clear();
  loadHtml();
  app = await import("./app.js");
  state = app.state;
});

beforeEach(() => {
  resetState();
  app.renderAll();
  const toast = document.querySelector("#toast");
  toast.textContent = "";
  toast.className = "toast";
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("utilities", () => {
  it("escapes unsafe html characters", () => {
    expect(app.escapeHtml(`&<>"'`)).toBe("&amp;&lt;&gt;&quot;&#039;");
    expect(app.escapeHtml(null)).toBe("");
  });

  it("maps status labels to classes", () => {
    expect(app.statusClass("healthy")).toBe("success");
    expect(app.statusClass("Review")).toBe("warning");
    expect(app.statusClass("Rejected")).toBe("danger");
    expect(app.statusClass("unknown")).toBe("neutral");
  });

  it("generates initials from a name", () => {
    expect(app.initials("Ada Lovelace")).toBe("AL");
    expect(app.initials("  single  ")).toBe("S");
    expect(app.initials("john ronald reuel")).toBe("JR");
  });

  it("normalizes project payloads from the api", () => {
    const normalized = app.normalizeProject({
      id: "p1",
      name: "Alpha",
      ownerName: "Mia Chen",
      environments: ["prod", { name: "staging" }],
      configCount: undefined,
      repoUrl: "",
      defaultFormat: "yaml",
    });

    expect(normalized.owner).toBe("Mia Chen");
    expect(normalized.environments).toEqual(["prod", "staging"]);
    expect(normalized.health).toBe("Review");
    expect(normalized.configCount).toBe(0);
    expect(normalized.lastChanged).toBe("live API");
  });
});

describe("api client", () => {
  it("adds auth headers and parses json responses", async () => {
    state.token = "token-123";
    const fetchSpy = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => JSON.stringify({ ok: true }),
    });
    vi.stubGlobal("fetch", fetchSpy);

    const data = await app.api("/projects");

    expect(fetchSpy).toHaveBeenCalledWith(
      "/api/v1/projects",
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer token-123",
          "Content-Type": "application/json",
        }),
      }),
    );
    expect(data).toEqual({ ok: true });
  });

  it("surfaces api error messages", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      text: async () => JSON.stringify({ message: "No access" }),
    }));

    await expect(app.api("/restricted")).rejects.toThrow("No access");
  });

  it("throws a descriptive error when the api is unreachable", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("network down")));

    await expect(app.api("/projects")).rejects.toThrow(
      "Cannot reach API /api/v1/projects.",
    );
  });
});

describe("rendering and state interactions", () => {
  it("switches views and updates navigation state", () => {
    state.activeView = "dashboard";
    app.renderNav();

    app.switchView("projects");

    expect(state.activeView).toBe("projects");
    expect(
      document.querySelector('#projectsView[data-view="projects"]').classList,
    ).toContain("active");
    expect(
      document.querySelector('#dashboardView[data-view="dashboard"]').classList,
    ).not.toContain("active");
    expect(
      document.querySelector('[data-view-target="projects"]').classList,
    ).toContain("active");
  });

  it("masks sensitive config values until revealed", () => {
    state.projects = [
      {
        id: "p1",
        name: "Alpha",
        owner: "Ops",
        defaultFormat: "json",
        configCount: 2,
        environments: ["prod"],
      },
    ];
    state.activeProjectId = "p1";
    state.configs = [
      {
        id: "c1",
        key: "db.password",
        value: "secret",
        type: "string",
        status: "Protected",
        updated: "admin",
        isSensitive: true,
        projectId: "p1",
        environment: "prod",
      },
      {
        id: "c2",
        key: "api.url",
        value: "https://example.com",
        type: "string",
        status: "Synced",
        updated: "admin",
        isSensitive: false,
        projectId: "p1",
        environment: "prod",
      },
    ];

    app.renderConfigRows();

    const rows = document.querySelector("#configRows").textContent;
    expect(rows).toContain("******");
    expect(rows).toContain("https://example.com");

    state.revealedKeys.add("p1:prod:db.password");
    app.renderConfigRows();
    expect(document.querySelector("#configRows").textContent).toContain(
      "secret",
    );
  });

  it("shows an empty message when no config rows match search", () => {
    state.projects = [
      {
        id: "p1",
        name: "Alpha",
        owner: "Ops",
        defaultFormat: "json",
        configCount: 1,
        environments: ["prod"],
      },
    ];
    state.activeProjectId = "p1";
    state.configs = [
      {
        id: "c1",
        key: "db.host",
        value: "db.prod",
        type: "string",
        status: "Synced",
        updated: "admin",
        isSensitive: false,
        projectId: "p1",
        environment: "prod",
      },
    ];
    state.configSearch = "not-found";

    app.renderConfigRows();

    expect(document.querySelector("#configRows").textContent).toContain(
      "No config keys match this view.",
    );
  });

  it("builds diff items and updates the diff panel", async () => {
    state.projects = [
      {
        id: "p1",
        name: "Alpha",
        owner: "Ops",
        defaultFormat: "json",
        configCount: 2,
        environments: ["staging", "prod"],
      },
    ];
    state.activeProjectId = "p1";

    const stagingEntries = [
      { key: "db.url", value: "staging", isSensitive: false },
      { key: "api.key", value: "secret", isSensitive: true },
    ];
    const prodEntries = [
      { key: "db.url", value: "prod", isSensitive: false },
      { key: "new.key", value: "value", isSensitive: false },
    ];

    vi.stubGlobal(
      "fetch",
      vi.fn((url) => {
        if (url.includes("env=staging")) {
          return Promise.resolve({
            ok: true,
            text: async () => JSON.stringify({ entries: stagingEntries }),
          });
        }
        if (url.includes("env=prod")) {
          return Promise.resolve({
            ok: true,
            text: async () => JSON.stringify({ entries: prodEntries }),
          });
        }
        return Promise.resolve({
          ok: true,
          text: async () => JSON.stringify({ entries: [] }),
        });
      }),
    );

    await app.loadDiff();

    expect(state.diffItems).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ key: "db.url", status: "modified" }),
        expect.objectContaining({ key: "api.key", status: "protected" }),
        expect.objectContaining({ key: "new.key", status: "removed" }),
      ]),
    );
    expect(document.querySelector("#validationBadge").textContent).toBe(
      "Review",
    );
    expect(document.querySelectorAll(".diff-item")).toHaveLength(3);
  });

  it("shows and hides toast messages on a timer", () => {
    vi.useFakeTimers();

    app.showToast("Saved");

    const toast = document.querySelector("#toast");
    expect(toast.textContent).toBe("Saved");
    expect(toast.classList.contains("show")).toBe(true);

    vi.advanceTimersByTime(2200);
    expect(toast.classList.contains("show")).toBe(false);

    vi.useRealTimers();
  });
});
