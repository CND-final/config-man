import { API_BASE, state } from "./state.js";

export async function api(path, options = {}) {
  const url = `${API_BASE}${path}`;
  let response;
  try {
    response = await fetch(url, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...(state.token ? { Authorization: `Bearer ${state.token}` } : {}),
        ...(options.headers || {}),
      },
    });
  } catch (error) {
    console.error("Network error while calling API:", url, error);
    throw new Error(
      `Cannot reach API ${url}. Check backend is running and configManApiBase is correct.`,
    );
  }

  const text = await response.text();
  let data = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch (error) {
      const preview = text.replace(/\s+/g, " ").trim().slice(0, 120);
      if (!response.ok) {
        throw new Error(
          `API ${path} failed: ${response.status}${preview ? ` ${preview}` : ""}`,
        );
      }
      throw new Error(
        `API ${path} returned invalid JSON${preview ? `: ${preview}` : ""}`,
      );
    }
  }
  if (!response.ok) {
    throw new Error(data?.message || `API request failed: ${response.status}`);
  }
  return data;
}
