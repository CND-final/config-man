import { describe, expect, it } from "vitest";
import {
  configFileForEntry,
  configsForActiveFile,
  ensureActiveConfigFile,
  normalizeConfigFileName,
} from "../configFiles.js";

describe("configFiles", () => {
  const files = [
    { id: "cfg-1", name: "primary.yaml" },
    { id: "cfg-2", name: "secondary.yaml" },
  ];

  it("selects config file for entry or falls back to first file", () => {
    expect(
      configFileForEntry({ configId: "cfg-2" }, { configFiles: files }),
    ).toEqual(files[1]);
    expect(
      configFileForEntry({ configId: "missing" }, { configFiles: files }),
    ).toEqual(files[0]);
    expect(configFileForEntry(null, { configFiles: [] })).toBeNull();
  });

  it("filters configs for active file", () => {
    const entries = [
      { id: "a", configId: "cfg-1" },
      { id: "b", configId: "cfg-2" },
      { id: "c", configId: "cfg-1" },
    ];
    expect(configsForActiveFile(entries, "cfg-1")).toEqual([
      entries[0],
      entries[2],
    ]);
    expect(configsForActiveFile(entries, "")).toEqual([]);
  });

  it("ensures active config file is valid", () => {
    const localState = { configFiles: files, activeConfigFile: "missing" };
    ensureActiveConfigFile(localState);
    expect(localState.activeConfigFile).toBe("cfg-1");

    const validState = { configFiles: files, activeConfigFile: "cfg-2" };
    ensureActiveConfigFile(validState);
    expect(validState.activeConfigFile).toBe("cfg-2");
  });

  it("normalizes config file names", () => {
    expect(normalizeConfigFileName(" app ")).toBe("app.yaml");
    expect(normalizeConfigFileName("service.yml")).toBe("service.yml");
    expect(normalizeConfigFileName("config.json")).toBe("config.json");
    expect(normalizeConfigFileName("")).toBe("");
  });
});
