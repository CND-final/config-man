const STANDARD_CONFIG_FILES = [
  {
    id: "application.yaml",
    name: "application.yaml",
    detail: "Application settings",
    matches: (entry) => !isRedisConfig(entry) && !isSecurityConfig(entry),
  },
  {
    id: "redis.yaml",
    name: "redis.yaml",
    detail: "Redis settings",
    matches: isRedisConfig,
  },
  {
    id: "security.json",
    name: "security.json",
    detail: "Security settings",
    matches: isSecurityConfig,
  },
];

export function configFilesForEntries(entries) {
  return STANDARD_CONFIG_FILES.map((file) => ({
    ...file,
    count: entries.filter((entry) => file.matches(entry)).length,
  }));
}

export function configFileForEntry(entry) {
  return (
    STANDARD_CONFIG_FILES.find((file) => file.matches(entry)) ||
    STANDARD_CONFIG_FILES[0]
  );
}

export function configsForActiveFile(entries, activeFileId) {
  const file =
    STANDARD_CONFIG_FILES.find((item) => item.id === activeFileId) ||
    STANDARD_CONFIG_FILES[0];
  return entries.filter((entry) => file.matches(entry));
}

export function ensureActiveConfigFile(state) {
  if (
    !STANDARD_CONFIG_FILES.some((file) => file.id === state.activeConfigFile)
  ) {
    state.activeConfigFile = STANDARD_CONFIG_FILES[0].id;
  }
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
