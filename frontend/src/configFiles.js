const FALLBACK_CONFIG_FILES = [
  fallbackFile("application.yaml", "Application settings", "application"),
  fallbackFile("redis.yaml", "Redis settings", "redis"),
  fallbackFile("security.json", "Security settings", "security"),
];

export function configFilesForEntries(entries, state = null) {
  return allConfigFiles(state).map((file) => ({
    ...file,
    count: entries.filter(
      (entry) => configFileForEntry(entry, state).id === file.id,
    ).length,
  }));
}

export function configFileForEntry(entry, state = null) {
  const files = allConfigFiles(state);
  if (entry?.configFileId) {
    const exact = files.find((file) => file.id === entry.configFileId);
    if (exact) return exact;
  }
  return files.find((file) => fileMatchesKey(file, entry?.key)) || files[0] || FALLBACK_CONFIG_FILES[0];
}

export function configsForActiveFile(entries, activeFileId, state = null) {
  return entries.filter(
    (entry) => configFileForEntry(entry, state).id === activeFileId,
  );
}

export function ensureActiveConfigFile(state) {
  if (!allConfigFiles(state).some((file) => file.id === state.activeConfigFile)) {
    state.activeConfigFile = allConfigFiles(state)[0]?.id || "";
  }
}

export function normalizeConfigFileName(name) {
  const trimmed = String(name || "").trim();
  if (!trimmed) return "";
  if (/\.(ya?ml|json|properties)$/i.test(trimmed)) return trimmed;
  return `${trimmed}.yaml`;
}

function allConfigFiles(state) {
  return Array.isArray(state?.configFiles) && state.configFiles.length
    ? state.configFiles
    : FALLBACK_CONFIG_FILES;
}

function fileMatchesKey(file, key) {
  key = String(key || "").trim().toLowerCase();
  const prefix = String(file.prefix || prefixFromFileName(file.name)).toLowerCase();
  if (!prefix) return false;
  if (prefix === "application") return true;
  if (key === prefix || key.startsWith(`${prefix}.`) || key.includes(`.${prefix}.`)) return true;
  if (prefix === "database" && (key === "db" || key.startsWith("db."))) return true;
  return false;
}

function fallbackFile(name, detail, prefix) {
  return {
    id: name,
    name,
    description: detail,
    detail,
    sourceType: "standard",
    prefix,
  };
}

function prefixFromFileName(name) {
  return String(name || "")
    .replace(/\.(ya?ml|json|properties)$/i, "")
    .trim()
    .toLowerCase();
}
