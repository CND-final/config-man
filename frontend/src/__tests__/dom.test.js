/* @vitest-environment jsdom */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { setAuthenticated, showToast } from "../dom.js";

describe("dom helpers", () => {
  beforeEach(() => {
    document.body.innerHTML = `
      <div id="toast"></div>
      <div id="loginScreen"></div>
      <div id="appShell" class="hidden"></div>
    `;
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows and hides toast messages", () => {
    vi.useFakeTimers();
    showToast("Saved!");
    const toast = document.querySelector("#toast");
    expect(toast.textContent).toBe("Saved!");
    expect(toast.classList.contains("show")).toBe(true);

    vi.advanceTimersByTime(2200);
    expect(toast.classList.contains("show")).toBe(false);
  });

  it("toggles authenticated layout", () => {
    const loginScreen = document.querySelector("#loginScreen");
    const appShell = document.querySelector("#appShell");

    setAuthenticated(true);
    expect(loginScreen.classList.contains("hidden")).toBe(true);
    expect(appShell.classList.contains("hidden")).toBe(false);

    setAuthenticated(false);
    expect(loginScreen.classList.contains("hidden")).toBe(false);
    expect(appShell.classList.contains("hidden")).toBe(true);
  });
});
