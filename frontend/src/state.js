export const API_BASE = "/api/v1";

export const state = {
  token: localStorage.getItem("configManToken") || "",
  user: JSON.parse(localStorage.getItem("configManUser") || "null"),
  activeView: "dashboard",
  activeProjectId: "",
  activeEnvironment: "prod",
  activeBranch: "default",
  activeConfigFile: "",
  projectDetailTab: "configs",
  projectMemberPickerOpen: false,
  projectMemberSearch: "",
  projectMemberSelection: new Set(),
  projectRoleMenuUserId: "",
  configMode: "view",
  compareSourceEnv: "dev",
  compareTargetEnv: "prod",
  compareConfigs: {},
  compareLoading: false,
  configCreateDrawerOpen: false,
  configFileCreateOpen: false,
  configFileMenuOpen: false,
  configSourceModalOpen: false,
  configSourceModalType: "",
  configFileSourceType: "blank",
  configFileSourceId: "",
  configFileDraftName: "",
  configFiles: [],
  newConfigValueType: "string",
  globalSearch: "",
  configSearch: "",
  inlineEdit: null,
  inlineSaving: false,
  pendingReviewChanges: [],
  configBaseline: new Map(),
  configBaselineContext: "",
  revealedKeys: new Set(),
  projects: [],
  configs: [],
  requests: [],
  requestDetailOpen: false,
  activeRequestId: "",
  templates: [],
  sharedConfigs: [],
  libraryTab: "templates",
  templateCreateModalOpen: false,
  sharedConfigCreateModalOpen: false,
  sharedConfigEditModalOpen: false,
  activeSharedConfigId: "",
  notifications: [],
  notificationPopoverOpen: false,
  templateModalOpen: false,
  activeTemplateId: "",
  templateValues: {},
  templateApplyFormat: "yaml",
  projectDraft: null,
  configHistory: [],
  historyLoading: false,
  reviewModalOpen: false,
  exportModalOpen: false,
  exportFormat: "yaml",
  importModalOpen: false,
  importPreview: null,
  importPreviewOpen: false,
  importApplying: false,
  projectModalOpen: false,
  historyModalOpen: false,
  userMenuOpen: false,
  groupModalOpen: false,
  groupLoading: false,
  groupError: "",
  groupMemberPickerOpen: false,
  groupMemberSearch: "",
  groupMemberSelection: new Set(),
  groupCreateOpen: false,
  groupCreateMemberSearch: "",
  groupCreateMemberSelection: new Set(),
  groupDetailTab: "members",
  groupRoleMenuUserId: "",
  groups: [],
  activeGroupId: "",
  activeGroup: null,
  users: [],
};

export const navItems = [
  { id: "dashboard", label: "Dashboard", icon: "/nav-dashboard.svg" },
  { id: "projects", label: "Projects", icon: "/nav-projects.svg" },
  { id: "templates", label: "Library", icon: "/nav-templates.svg" },
  { id: "requests", label: "Requests", icon: "/nav-requests.svg" },
];

export function projectBranches(project) {
  const branches = project?.branches || project?.Branches || [];
  const names = branches
    .map((branch) =>
      typeof branch === "string" ? branch : branch.name || branch.Name || branch.id || branch.ID,
    )
    .map((branch) => String(branch || "").trim().toLowerCase())
    .filter(Boolean);
  return names.length ? Array.from(new Set(names)) : ["default"];
}

export function defaultBranchForProject(project) {
  const branches = projectBranches(project);
  return branches.includes("default") ? "default" : branches[0];
}

export function activeProject() {
  return (
    state.projects.find((project) => project.id === state.activeProjectId) ||
    state.projects[0]
  );
}

export function normalizeProject(project) {
  const environments = project.environments || project.Environments || [];
  const branches = projectBranches(project);
  return {
    ...project,
    id: project.id || project.ID,
    name: project.name || project.Name,
    groupId: project.groupId || project.GroupID || "",
    environments: environments.map((environment) =>
      typeof environment === "string" ? environment : environment.name,
    ),
    branches,
    lastChanged: "live API",
    configCount: project.configCount ?? 0,
  };
}
