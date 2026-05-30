/* @vitest-environment jsdom */
import { afterEach, describe, expect, it } from "vitest";
import { renderNav, renderNotifications } from "../render.js";
import { state } from "../state.js";

afterEach(() => {
  state.user = null;
  state.activeView = "dashboard";
  state.notificationPopoverOpen = false;
  state.notifications = [];
  document.body.innerHTML = "";
});

describe("render", () => {
  it("hides requests nav item for viewers", () => {
    document.body.innerHTML = `<div id="navList"></div>`;
    state.user = { role: "viewer" };
    state.activeView = "dashboard";

    renderNav();

    const nav = document.querySelector("#navList").innerHTML;
    expect(nav).not.toContain('data-view-target="requests"');
  });

  it("shows requests nav item for non-viewers", () => {
    document.body.innerHTML = `<div id="navList"></div>`;
    state.user = { role: "developer" };
    state.activeView = "requests";

    renderNav();

    const nav = document.querySelector("#navList").innerHTML;
    expect(nav).toContain('data-view-target="requests"');
    expect(nav).toContain('nav-item active');
  });

  it("renders notification count and popover state", () => {
    document.body.innerHTML = `
      <button id="notificationButton"></button>
      <span id="notificationCount" class="hidden"></span>
      <div id="notificationPopover" class="hidden"></div>
    `;
    state.notificationPopoverOpen = true;
    state.notifications = [
      {
        id: "n1",
        title: "Update",
        message: "Config changed",
        createdAt: "2026-01-01T00:00:00Z",
        read: false,
      },
      { id: "n2", read: true },
    ];

    renderNotifications();

    const count = document.querySelector("#notificationCount");
    const button = document.querySelector("#notificationButton");
    const popover = document.querySelector("#notificationPopover");

    expect(count.textContent).toBe("1");
    expect(count.classList.contains("hidden")).toBe(false);
    expect(button.getAttribute("aria-expanded")).toBe("true");
    expect(popover.classList.contains("hidden")).toBe(false);
    expect(popover.innerHTML).toContain("Notifications");
  });
});
