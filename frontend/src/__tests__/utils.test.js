import { describe, expect, it, vi } from "vitest";
import {
  escapeHtml,
  formatDateTime,
  initials,
  isProdSensitiveEdit,
  parseEnvironmentInput,
  statusClass,
} from "../utils.js";

describe("utils", () => {
  it("escapes HTML special characters", () => {
    expect(escapeHtml(`5 > 3 & 2 < 4 "ok" 'no'`)).toBe(
      "5 &gt; 3 &amp; 2 &lt; 4 &quot;ok&quot; &#039;no&#039;",
    );
    expect(escapeHtml(null)).toBe("");
  });

  it("maps status values to classes", () => {
    expect(statusClass("Approved")).toBe("success");
    expect(statusClass("warning")).toBe("warning");
    expect(statusClass("FAILED")).toBe("danger");
    expect(statusClass("unknown")).toBe("neutral");
  });

  it("builds initials from names", () => {
    expect(initials("Ada Lovelace")).toBe("AL");
    expect(initials("  single  ")).toBe("S");
    expect(initials("alpha beta gamma")).toBe("AB");
  });

  it("formats dates or falls back when missing", () => {
    expect(formatDateTime("")).toBe("unknown time");
    const formatter = vi
      .spyOn(Intl, "DateTimeFormat")
      .mockImplementation(function () {
        return { format: () => "Jan 1, 2024, 00:00" };
      });
    try {
      expect(formatDateTime("2024-01-01T00:00:00Z")).toBe("Jan 1, 2024, 00:00");
    } finally {
      formatter.mockRestore();
    }
  });

  it("parses environments from comma-separated input", () => {
    expect(parseEnvironmentInput("Prod, Staging , ,DEV")).toEqual([
      "prod",
      "staging",
      "dev",
    ]);
  });

  it("detects sensitive prod edits", () => {
    expect(
      isProdSensitiveEdit({
        environment: "prod",
        isSensitive: true,
        key: "DB_PASSWORD",
      }),
    ).toBe(true);
    expect(
      isProdSensitiveEdit({
        environment: "prod",
        isSensitive: false,
        key: "api_token",
      }),
    ).toBe(true);
    expect(
      isProdSensitiveEdit({
        environment: "dev",
        isSensitive: true,
        key: "DB_PASSWORD",
      }),
    ).toBe(false);
  });
});
