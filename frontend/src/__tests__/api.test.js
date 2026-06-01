/* @vitest-environment jsdom */
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "../api.js";
import { state } from "../state.js";

afterEach(() => {
  state.token = "";
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("api", () => {
  it("adds auth header and parses JSON responses", async () => {
    state.token = "token-123";
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: () => Promise.resolve('{"ok":true}'),
    });
    vi.stubGlobal("fetch", fetchMock);

    const data = await api("/health");
    expect(data).toEqual({ ok: true });
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/health", {
      headers: {
        "Content-Type": "application/json",
        Authorization: "Bearer token-123",
      },
    });
  });

  it("throws server-provided error messages", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      text: () => Promise.resolve('{"message":"bad request"}'),
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(api("/projects")).rejects.toThrow("bad request");
  });

  it("throws when JSON is invalid", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: () => Promise.resolve("not-json"),
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(api("/templates")).rejects.toThrow(
      "API /templates returned invalid JSON",
    );
  });

  it("surfaces network failures", async () => {
    const fetchMock = vi.fn().mockRejectedValue(new Error("network down"));
    vi.stubGlobal("fetch", fetchMock);
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    await expect(api("/projects")).rejects.toThrow(
      "Cannot reach API /api/v1/projects.",
    );
    expect(consoleSpy).toHaveBeenCalled();
  });
});
