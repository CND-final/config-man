export function configFileForEntry(entry, state = null) {
  const files = allConfigFiles(state);
  if (entry?.configId) {
    const exact = files.find((file) => file.id === entry.configId);
    if (exact) return exact;
  }
  return files[0] || null;
}

export function configsForActiveFile(entries, activeFileId) {
  if (!activeFileId) return [];
  return entries.filter((entry) => entry.configId === activeFileId);
}

export function ensureActiveConfigFile(state) {
  const files = allConfigFiles(state);
  if (!files.some((file) => file.id === state.activeConfigFile)) {
    state.activeConfigFile = files[0]?.id || "";
  }
}

export function normalizeConfigFileName(name) {
  const trimmed = String(name || "").trim();
  if (!trimmed) return "";
  if (/\.(ya?ml|json|properties)$/i.test(trimmed)) return trimmed;
  return `${trimmed}.yaml`;
}

function allConfigFiles(state) {
  return Array.isArray(state?.configFiles) ? state.configFiles : [];
}
