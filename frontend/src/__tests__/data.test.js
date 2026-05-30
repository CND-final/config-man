/* @vitest-environment jsdom */
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("../api.js", () => ({
  api: vi.fn(),
}));

import { api } from "../api.js";
import { normalizeGroup, reloadGroups } from "../data.js";
import { state } from "../state.js";

afterEach(() => {
  vi.clearAllMocks();
  state.groups = [];
  state.activeGroupId = "";
  state.activeGroup = null;
  state.groupError = "";
});

describe("data helpers", () => {
  it("normalizes groups with mixed payload shapes", () => {
    const group = normalizeGroup({
      ID: "grp-1",
      Name: "Core Team",
      groupMembers: [{ id: "alice" }],
      managedProjects: ["project-a", { id: "project-b", name: "Project B" }],
      memberCount: 3,
    });

    expect(group.id).toBe("grp-1");
    expect(group.name).toBe("Core Team");
    expect(group.memberCount).toBe(3);
    expect(group.members).toEqual([{ id: "alice" }]);
    expect(group.projects).toEqual([
      { id: "project-a", name: "project-a", environments: [] },
      expect.objectContaining({ id: "project-b", name: "Project B" }),
    ]);
  });

  it("reloads groups and updates active selection", async () => {
    api.mockResolvedValueOnce({
      groups: [{ id: "g1", name: "Group 1", members: [] }],
    });
    state.activeGroupId = "old";

    await reloadGroups();

    expect(state.groups[0].id).toBe("g1");
    expect(state.activeGroupId).toBe("g1");
    expect(state.groupError).toBe("");
  });

  it("silently handles group load errors when requested", async () => {
    api.mockRejectedValueOnce(new Error("boom"));
    state.groups = [{ id: "keep" }];
    state.activeGroupId = "keep";

    await expect(reloadGroups({ silent: true })).resolves.toBeUndefined();

    expect(state.groups).toEqual([]);
    expect(state.activeGroupId).toBe("");
    expect(state.groupError).toBe("");
  });
});
