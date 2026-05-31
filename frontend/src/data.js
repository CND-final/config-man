import { api } from "./api.js";
import { ensureActiveConfigFile } from "./configFiles.js";
import { activeProject, defaultBranchForProject, normalizeProject, projectBranches, state } from "./state.js";

export async function loadInitialData() {
  const [projects, templates, sharedConfigs, notifications, requests] =
    await Promise.all([
      api("/projects"),
      api("/templates"),
      api("/shared-configs"),
      api("/notifications"),
      api("/review-requests"),
    ]);

  state.projects = listFromPayload(projects, "projects").map(normalizeProject);
  state.templates = normalizeTemplates(listFromPayload(templates, "templates"));
  state.sharedConfigs = normalizeSharedConfigs(listFromPayload(sharedConfigs, "sharedConfigs"));
  state.notifications = listFromPayload(notifications, "notifications");
  state.requests = listFromPayload(requests, "requests");
  await reloadGroups({ silent: true });

  if (!state.activeProjectId && state.projects[0]) {
    state.activeProjectId = state.projects[0].id;
  }
  const project = activeProject();
  if (project && !project.environments.includes(state.activeEnvironment)) {
    state.activeEnvironment = project.environments[0];
  }
  if (project && !projectBranches(project).includes(state.activeBranch)) {
    state.activeBranch = defaultBranchForProject(project);
  }

  await loadConfigsAndHistory();
}

function normalizeTemplates(templates) {
  return templates.map((template) => ({
    ...template,
    itemType: "template",
    format: template.format || "base",
    keys: template.entries?.map((entry) => entry.key) || [],
    variables: template.variables || [],
  }));
}

function normalizeSharedConfigs(items) {
  return items.map((item) => {
    const entries = item.entries || [];
    return {
      ...item,
      itemType: "shared_config",
      format: item.format || "yaml",
      entries,
      keys: Array.from(new Set(entries.map((entry) => entry.key))),
      affectedProjects: item.affectedProjects || [],
    };
  });
}

export async function reloadNotifications() {
  state.notifications = listFromPayload(await api("/notifications"), "notifications");
}


export async function reloadProjectMembers(projectId = activeProject()?.id) {
  if (!projectId) return [];
  const payload = await api(`/projects/${encodeURIComponent(projectId)}/members`);
  const members = listFromPayload(payload, "members");
  state.projects = state.projects.map((project) =>
    project.id === projectId
      ? { ...project, members, memberCount: members.length }
      : project,
  );
  return members;
}

export async function reloadProjects() {
  const projects = await api("/projects");
  state.projects = listFromPayload(projects, "projects").map(normalizeProject);
}

export async function reloadTemplates() {
  const templates = await api("/templates");
  state.templates = normalizeTemplates(listFromPayload(templates, "templates"));
}

export async function reloadSharedConfigs() {
  const sharedConfigs = await api("/shared-configs");
  state.sharedConfigs = normalizeSharedConfigs(listFromPayload(sharedConfigs, "sharedConfigs"));
}

export async function loadConfigs(revealSensitive = false) {
  const project = activeProject();
  if (!project) {
    state.configs = [];
    state.configHistory = [];
    state.configBaseline = new Map();
    state.configBaselineContext = "";
    state.pendingReviewChanges = [];
    return;
  }

  const data = await api(
    `/projects/${project.id}/configs?env=${encodeURIComponent(
      state.activeEnvironment,
    )}&branch=${encodeURIComponent(state.activeBranch)}${revealSensitive ? "&revealSensitive=true" : ""}`,
  );
  state.configFiles = configsFromPayload(data);
  ensureActiveConfigFile(state);
  state.configs = entriesFromConfigPayload(data).map((entry) => ({
    ...entry,
    updated: entry.updatedBy,
  }));

  const context = `${project.id}:${state.activeBranch}:${state.activeEnvironment}:${state.activeConfigFile}`;
  if (state.configBaselineContext !== context) {
    state.configBaselineContext = context;
    state.configBaseline = new Map(
      state.configs.map((entry) => [entry.id, baselineConfig(entry)]),
    );
    state.pendingReviewChanges = [];
  }
}

function configsFromPayload(data) {
  const configs = Array.isArray(data?.configs) ? data.configs : data?.files || [];
  return configs.map((config) => ({
    ...config,
    entries: (config.entries || []).map((entry) => ({
      ...entry,
      updated: entry.updatedBy,
    })),
  }));
}

function entriesFromConfigPayload(data) {
  if (Array.isArray(data?.configs)) {
    return data.configs.flatMap((config) => config.entries || []);
  }
  return data?.entries || [];
}

function baselineConfig(entry) {
  return {
    id: entry.id,
    key: entry.key,
    value: entry.value,
    environment: entry.environment,
    branch: entry.branch || state.activeBranch,
  };
}

export async function loadConfigsAndHistory(revealSensitive = false) {
  await loadConfigs(revealSensitive);
  await loadConfigHistory();
}

export async function loadCompareConfigs() {
  const project = activeProject();
  if (!project) {
    state.compareConfigs = {};
    return;
  }

  const environments = Array.from(
    new Set([state.compareSourceEnv, state.compareTargetEnv].filter(Boolean)),
  );
  if (!environments.length) {
    state.compareConfigs = {};
    return;
  }

  state.compareLoading = true;
  try {
    const results = await Promise.all(
      environments.map(async (environment) => {
        const data = await api(
          "/projects/" +
            project.id +
            "/configs?env=" +
            encodeURIComponent(environment) +
            "&branch=" +
            encodeURIComponent(state.activeBranch),
        );
        if (environment === state.activeEnvironment) {
          state.configFiles = configsFromPayload(data) || state.configFiles;
        }
        return [
          environment,
          entriesFromConfigPayload(data).map((entry) => ({
            ...entry,
            updated: entry.updatedBy,
          })),
        ];
      }),
    );
    state.compareConfigs = Object.fromEntries(results);
  } finally {
    state.compareLoading = false;
  }
}

export async function loadConfigHistory() {
  const project = activeProject();
  if (!project) {
    state.configHistory = [];
    return;
  }

  const data = await api(
    `/projects/${project.id}/config-history?env=${encodeURIComponent(state.activeEnvironment)}&branch=${encodeURIComponent(state.activeBranch)}`,
  );
  state.configHistory = data.revisions || [];
}

function listFromPayload(payload, key) {
  if (Array.isArray(payload)) return payload;
  if (Array.isArray(payload?.[key])) return payload[key];
  return [];
}

export function normalizeGroup(group) {
  const members =
    group.members || group.groupMembers || group.GroupMembers || [];
  const rawProjects =
    group.projects || group.managedProjects || group.Projects || [];
  const projects = rawProjects.map((project) =>
    typeof project === "string"
      ? { id: project, name: project, environments: [] }
      : normalizeProject(project),
  );
  return {
    ...group,
    id: group.id || group.groupId || group.ID,
    name: group.name || group.groupName || group.Name,
    memberCount: group.memberCount ?? members.length ?? 0,
    projectCount: group.projectCount ?? projects.length ?? 0,
    members,
    projects,
  };
}

export async function reloadGroups(options = {}) {
  try {
    const payload = await api("/groups");
    state.groups = listFromPayload(payload, "groups").map(normalizeGroup);
    state.groupError = "";
    if (!state.groups.some((group) => group.id === state.activeGroupId)) {
      state.activeGroupId = state.groups[0]?.id || "";
      state.activeGroup = null;
    }
  } catch (error) {
    state.groups = [];
    state.activeGroupId = "";
    state.activeGroup = null;
    state.groupError = options.silent ? "" : error.message;
    if (!options.silent) throw error;
  }
}

export async function reloadGroupDetail(groupId = state.activeGroupId) {
  if (!groupId) {
    state.activeGroup = null;
    return;
  }
  const payload = await api(`/groups/${encodeURIComponent(groupId)}`);
  const group = payload?.group || payload;
  if (!group?.id && !group?.groupId && !group?.ID) {
    state.activeGroup =
      state.groups.find((item) => item.id === groupId) || null;
    return;
  }
  state.activeGroup = normalizeGroup(group);
  state.activeGroupId = state.activeGroup.id;
}

export async function reloadUsers(options = {}) {
  try {
    const payload = await api("/users");
    state.users = listFromPayload(payload, "users");
  } catch (error) {
    state.users = [];
    if (!options.silent) throw error;
  }
}
