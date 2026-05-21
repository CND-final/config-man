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
  const data = text ? JSON.parse(text) : null;
  if (!response.ok) {
    throw new Error(data?.message || `API request failed: ${response.status}`);
  }
  return data;
}
