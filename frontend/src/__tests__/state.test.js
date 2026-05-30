import { afterEach, describe, expect, it } from "vitest";
import { activeProject, normalizeProject, state } from "../state.js";

const originalState = {
  projects: state.projects,
  activeProjectId: state.activeProjectId,
};

afterEach(() => {
  state.projects = originalState.projects;
  state.activeProjectId = originalState.activeProjectId;
});

describe("state helpers", () => {
  it("returns the active project when selected", () => {
    state.projects = [
      { id: "alpha", name: "Alpha" },
      { id: "beta", name: "Beta" },
    ];
    state.activeProjectId = "beta";
    expect(activeProject()).toEqual(state.projects[1]);
  });

  it("falls back to the first project when none selected", () => {
    state.projects = [
      { id: "alpha", name: "Alpha" },
      { id: "beta", name: "Beta" },
    ];
    state.activeProjectId = "missing";
    expect(activeProject()).toEqual(state.projects[0]);
  });

  it("normalizes project payloads from the API", () => {
    const normalized = normalizeProject({
      ID: "p-1",
      Name: "Billing",
      GroupID: "grp-1",
      Environments: [{ name: "prod" }, { name: "dev" }],
      configCount: 4,
    });
    expect(normalized.id).toBe("p-1");
    expect(normalized.name).toBe("Billing");
    expect(normalized.groupId).toBe("grp-1");
    expect(normalized.environments).toEqual(["prod", "dev"]);
    expect(normalized.configCount).toBe(4);
    expect(normalized.lastChanged).toBe("live API");
  });
});
