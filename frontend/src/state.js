export const API_BASE = "/api/v1";

export const state = {
  token: localStorage.getItem("configManToken") || "",
  user: JSON.parse(localStorage.getItem("configManUser") || "null"),
  activeView: "dashboard",
  activeProjectId: "",
  activeEnvironment: "prod",
  activeConfigFile: "application.yaml",
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
  templates: [],
  templateCreateModalOpen: false,
  templatePickerActive: false,
  templateModalOpen: false,
  activeTemplateId: "",
  templateValues: {},
  templateApplyFormat: "yaml",
  projectDraft: null,
  projectTemplateSelection: null,
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
  { id: "templates", label: "Templates", icon: "/nav-templates.svg" },
  { id: "requests", label: "Requests", icon: "/nav-requests.svg" },
];

export function activeProject() {
  return (
    state.projects.find((project) => project.id === state.activeProjectId) ||
    state.projects[0]
  );
}

export function normalizeProject(project) {
  return {
    ...project,
    owner: project.ownerName,
    environments: project.environments.map((environment) =>
      typeof environment === "string" ? environment : environment.name,
    ),
    lastChanged: "live API",
    configCount: project.configCount ?? 0,
  };
}
