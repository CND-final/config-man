const STANDARD_CONFIG_FILES = [
  {
    id: "application.yaml",
    name: "application.yaml",
    detail: "Application settings",
    kind: "standard",
  },
  {
    id: "redis.yaml",
    name: "redis.yaml",
    detail: "Redis settings",
    kind: "standard",
    matches: isRedisConfig,
  },
  {
    id: "security.json",
    name: "security.json",
    detail: "Security settings",
    kind: "standard",
    matches: isSecurityConfig,
  },
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
  const customFile = customConfigFiles(state).find((file) =>
    customFileMatches(file, entry),
  );
  if (customFile) return customFile;

  const standardFile = STANDARD_CONFIG_FILES.find((file) =>
    file.matches?.(entry),
  );
  if (standardFile) return standardFile;

  return STANDARD_CONFIG_FILES[0];
}

export function configsForActiveFile(entries, activeFileId, state = null) {
  return entries.filter(
    (entry) => configFileForEntry(entry, state).id === activeFileId,
  );
}

export function ensureActiveConfigFile(state) {
  if (
    !allConfigFiles(state).some((file) => file.id === state.activeConfigFile)
  ) {
    state.activeConfigFile = STANDARD_CONFIG_FILES[0].id;
  }
}

export function saveCustomConfigFiles(state) {
  const projectId = state.activeProjectId || "global";
  localStorage.setItem(
    customFileStorageKey(projectId),
    JSON.stringify(state.customConfigFiles || []),
  );
}

export function loadCustomConfigFiles(state) {
  const projectId = state.activeProjectId || "global";
  try {
    const raw = localStorage.getItem(customFileStorageKey(projectId));
    state.customConfigFiles = raw ? JSON.parse(raw) : [];
  } catch (error) {
    state.customConfigFiles = [];
  }
}

export function normalizeConfigFileName(name) {
  const trimmed = String(name || "").trim();
  if (!trimmed) return "";
  if (/\.(ya?ml|json|properties)$/i.test(trimmed)) return trimmed;
  return `${trimmed}.yaml`;
}

export function newCustomConfigFile(name, source = {}) {
  const fileName = normalizeConfigFileName(name);
  const id = fileName.toLowerCase();
  return {
    id,
    name: fileName,
    detail: source.label || "Custom config file",
    kind: "custom",
    sourceType: source.type || "blank",
    sourceId: source.id || "",
    prefix: prefixFromFileName(fileName),
  };
}

function allConfigFiles(state) {
  return [...customConfigFiles(state), ...STANDARD_CONFIG_FILES];
}

function customConfigFiles(state) {
  return Array.isArray(state?.customConfigFiles) ? state.customConfigFiles : [];
}

function customFileMatches(file, entry) {
  const key = normalizedKey(entry);
  const prefix = String(
    file.prefix || prefixFromFileName(file.name),
  ).toLowerCase();
  if (!prefix) return false;
  if (key === prefix || key.startsWith(`${prefix}.`)) return true;
  if (prefix === "database" && (key === "db" || key.startsWith("db.")))
    return true;
  return false;
}

function prefixFromFileName(name) {
  return String(name || "")
    .replace(/\.(ya?ml|json|properties)$/i, "")
    .trim()
    .toLowerCase();
}

function customFileStorageKey(projectId) {
  return `configManConfigFiles:${projectId}`;
}

function isRedisConfig(entry) {
  const key = normalizedKey(entry);
  return key === "redis" || key.startsWith("redis.") || key.includes(".redis.");
}

function isSecurityConfig(entry) {
  const key = normalizedKey(entry);
  return [
    "security",
    "auth",
    "jwt",
    "oauth",
    "cors",
    "tls",
    "ssl",
    "saml",
    "session",
  ].some(
    (prefix) =>
      key === prefix ||
      key.startsWith(`${prefix}.`) ||
      key.includes(`.${prefix}.`),
  );
}

function normalizedKey(entry) {
  return String(entry?.key || "")
    .trim()
    .toLowerCase();
}
