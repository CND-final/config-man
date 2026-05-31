import { api } from "./api.js";
import { $, $all, showToast } from "./dom.js";
import {
  loadCompareConfigs,
  loadConfigHistory,
  loadConfigsAndHistory,
  reloadGroupDetail,
  reloadGroups,
  reloadProjectMembers,
  reloadProjects,
  reloadTemplates,
  reloadSharedConfigs,
  reloadNotifications,
  reloadUsers,
} from "./data.js";
import {
  renderAll,
  renderConfigCreateDrawer,
  renderConfigRow,
  renderConfigRows,
  renderDashboard,
  renderExportModal,
  renderNav,
  renderNotifications,
  renderImportPreview,
  renderGroupCreateMemberPicker,
  renderGroupMemberPicker,
  renderGroupModal,
  renderProjectFormOptions,
  renderProjectMemberPicker,
  renderProjectMembersPanel,
  renderRequests,
  renderRequestDetailModal,
  renderReviewModal,
  renderReviewDock,
  renderTemplateCreateModal,
  renderSharedConfigCreateModal,
  renderSharedConfigEditModal,
  renderTemplateModal,
  renderUserMenu,
  renderVersionHistory,
} from "./render.js";
import { activeProject, defaultBranchForProject, projectBranches, state } from "./state.js";
import { parseEnvironmentInput } from "./utils.js";


function clearProjectMemberPicker() {
  state.projectMemberPickerOpen = false;
  state.projectMemberSearch = "";
  state.projectMemberSelection = new Set();
  state.projectRoleMenuUserId = "";
}

export function canEditProjectMembers(project = activeProject()) {
  if (!project) return false;
  if (state.user?.role === "system_admin") return true;
  if (state.user?.role === "group_admin") {
    const group = state.groups.find((item) => item.id === project.groupId);
    return (group?.members || []).some((member) => {
      const id = member.id || member.userId;
      return id === state.user?.id && member.groupRole === "group_admin";
    });
  }
  return (project.members || []).some((member) => {
    const id = member.id || member.userId;
    return id === state.user?.id && member.projectRole === "project_admin";
  });
}

export async function setProjectDetailTab(tab) {
  if (!["configs", "members"].includes(tab)) return;
  state.projectDetailTab = tab;
  clearProjectMemberPicker();
  if (tab === "members") {
    await Promise.all([
      reloadProjectMembers(activeProject()?.id),
      reloadUsers({ silent: true }),
      reloadGroups({ silent: true }),
    ]);
  }
  renderConfigRows();
}

export function toggleProjectMemberPicker(open = !state.projectMemberPickerOpen) {
  state.projectMemberPickerOpen = open;
  state.projectMemberSearch = "";
  state.projectMemberSelection = new Set();
  state.projectRoleMenuUserId = "";
  renderProjectMembersPanel();
}

export function updateProjectMemberSearch(value) {
  state.projectMemberSearch = value;
  renderPreservingMemberPicker(
    "projectMemberTools",
    "projectMemberSearch",
    renderProjectMemberPicker,
  );
}

export function toggleProjectMemberSelection(userId, selected) {
  if (!userId) return;
  if (selected) {
    state.projectMemberSelection.add(userId);
  } else {
    state.projectMemberSelection.delete(userId);
  }
  renderPreservingMemberPicker(
    "projectMemberTools",
    "projectMemberSearch",
    renderProjectMemberPicker,
  );
}

export function removeSelectedProjectMember(userId) {
  state.projectMemberSelection.delete(userId);
  renderPreservingMemberPicker(
    "projectMemberTools",
    "projectMemberSearch",
    renderProjectMemberPicker,
  );
}

function projectMemberPayload(project, overrides = new Map()) {
  return (project?.members || []).map((member) => {
    const id = member.id || member.userId;
    return {
      userId: id,
      projectRole: overrides.get(id) || member.projectRole || "viewer",
    };
  });
}

async function saveProjectMembers(members) {
  const project = activeProject();
  if (!project) return [];
  const payload = await api(`/projects/${encodeURIComponent(project.id)}/members`, {
    method: "PUT",
    body: JSON.stringify({ members }),
  });
  const updatedMembers = payload.members || [];
  state.projects = state.projects.map((item) =>
    item.id === project.id
      ? { ...item, members: updatedMembers, memberCount: updatedMembers.length }
      : item,
  );
  return updatedMembers;
}

export async function addProjectMembers() {
  const project = activeProject();
  if (!canEditProjectMembers(project)) {
    showToast("Only group admins or project admins can edit project members");
    return;
  }
  const selected = Array.from(state.projectMemberSelection);
  if (!project || selected.length === 0) return;
  const existing = projectMemberPayload(project);
  const existingIds = new Set(existing.map((member) => member.userId));
  const additions = selected
    .filter((userId) => !existingIds.has(userId))
    .map((userId) => ({ userId, projectRole: "viewer" }));
  await saveProjectMembers([...existing, ...additions]);
  clearProjectMemberPicker();
  renderConfigRows();
  showToast(`${additions.length} member${additions.length === 1 ? "" : "s"} added`);
}

export async function removeProjectMember(userId) {
  const project = activeProject();
  if (!canEditProjectMembers(project) || !project || !userId) return;
  const members = projectMemberPayload(project).filter((member) => member.userId !== userId);
  await saveProjectMembers(members);
  renderConfigRows();
  showToast("Member removed");
}

export function toggleProjectRoleMenu(userId) {
  state.projectRoleMenuUserId =
    state.projectRoleMenuUserId === userId ? "" : userId;
  renderProjectMembersPanel();
}

export async function updateProjectMemberRole(userId, projectRole) {
  const project = activeProject();
  if (!canEditProjectMembers(project) || !project || !userId) return;
  const members = projectMemberPayload(project, new Map([[userId, projectRole]]));
  await saveProjectMembers(members);
  state.projectRoleMenuUserId = "";
  renderConfigRows();
  showToast("Project role updated");
}


function projectRoleForCurrentUser(project = activeProject()) {
  if (!project) return "";
  if (state.user?.role === "system_admin") return "system_admin";
  const member = (project.members || []).find((item) => {
    const id = item.id || item.userId;
    return id === state.user?.id;
  });
  return member?.projectRole || "";
}

function canDirectWriteEnvironment(project, environment) {
  const role = projectRoleForCurrentUser(project);
  if (role === "system_admin" || role === "project_admin") return true;
  return role === "developer" && String(environment).toLowerCase() !== "prod";
}

function canCreateProjectReview(project = activeProject()) {
  const role = projectRoleForCurrentUser(project);
  return ["system_admin", "project_admin", "developer", "reviewer"].includes(role);
}

function shouldStageReviewChange(project, environment) {
  return (
    String(environment).toLowerCase() === "prod" &&
    !canDirectWriteEnvironment(project, environment) &&
    canCreateProjectReview(project)
  );
}

function stageLocalConfigChange(configId, field, value) {
  state.configs = state.configs.map((entry) => {
    if (entry.id !== configId) return entry;
    const sharedValue = sharedDefaultValueForEntry(entry);
    const hasSharedSource = entry.sourceType === "shared-config" || sharedValue !== undefined;
    const nextEntry = {
      ...entry,
      [field]: value,
      inherited: false,
    };
    if (hasSharedSource) {
      nextEntry.sharedValue = sharedValue ?? "";
      nextEntry.overridden = sharedEntryHasLocalChange(nextEntry);
    } else {
      nextEntry.overridden = entry.overridden;
    }
    return nextEntry;
  });
  syncPendingReviewChanges();
}

function sharedDefaultValueForEntry(entry) {
  if (Object.prototype.hasOwnProperty.call(entry, "sharedValue")) {
    if (entry.inherited && entry.sharedValue === "" && entry.value !== "") {
      return entry.value;
    }
    return entry.sharedValue;
  }
  if (entry.inherited || entry.sourceType === "shared-config") {
    return entry.value;
  }
  return undefined;
}

function sharedEntryHasLocalChange(entry) {
  if (!Object.prototype.hasOwnProperty.call(entry, "sharedValue")) return false;
  return String(entry.value ?? "") !== String(entry.sharedValue ?? "");
}

function isSharedConfigEntry(config) {
  return config.inherited || config.sourceType === "shared-config" || Boolean(config.sourceId);
}

function stageLocalConfigCreate(project, request, bodyBase) {
  const branch = request.branch || state.activeBranch;
  const id = `draft-${project.id}-${branch}-${request.environment}-${bodyBase.key}`.replace(/[^a-z0-9_-]+/gi, "-");
  const existing = state.configs.find(
    (entry) =>
      (entry.branch || state.activeBranch) === branch &&
      entry.environment === request.environment &&
      entry.key === bodyBase.key &&
      entry.configId === bodyBase.configId,
  );
  if (existing) {
    stageLocalConfigChange(existing.id, "value", request.value);
    return;
  }
  state.configs = [
    ...state.configs,
    {
      id,
      projectId: project.id,
      environment: request.environment,
      branch,
      configId: bodyBase.configId,
      key: bodyBase.key,
      value: request.value,
      valueType: bodyBase.valueType,
      isSensitive: bodyBase.isSensitive,
    },
  ];
  syncPendingReviewChanges();
}

export function switchView(viewId) {
  state.activeView = viewId;
  $all(".view").forEach((view) => {
    view.classList.toggle("active", view.dataset.view === viewId);
  });
  renderNav();
}

export function setLibraryTab(tab) {
  if (!["templates", "shared-config"].includes(tab)) return;
  state.libraryTab = tab;
  renderAll();
}

export function canCreateGroup() {
  return ["system_admin", "group_admin"].includes(state.user?.role);
}

export function canEditGroupMembers() {
  return ["system_admin", "group_admin"].includes(state.user?.role);
}

export function manageableSharedConfigGroups() {
  if (state.user?.role === "system_admin") return state.groups;
  return state.groups.filter((group) =>
    (group.members || []).some((member) => {
      const id = member.id || member.userId;
      return id === state.user?.id && member.groupRole === "group_admin";
    }),
  );
}

export function canCreateSharedConfig() {
  return state.user?.role === "system_admin" || manageableSharedConfigGroups().length > 0;
}

export function canManageSharedConfig(item) {
  if (state.user?.role === "system_admin") return true;
  if (item?.scope !== "group") return false;
  return manageableSharedConfigGroups().some((group) => group.id === item.scopeId);
}

function clearGroupMemberPicker() {
  state.groupMemberPickerOpen = false;
  state.groupMemberSearch = "";
  state.groupMemberSelection = new Set();
  state.groupRoleMenuUserId = "";
}

function clearGroupCreatePicker() {
  state.groupCreateMemberSearch = "";
  state.groupCreateMemberSelection = new Set();
}

function renderPreservingMemberPicker(containerId, inputId, renderFn) {
  const container = $(`#${containerId}`);
  const optionsScrollTop =
    container?.querySelector(".member-picker-options")?.scrollTop ?? 0;
  const selectedScrollTop =
    container?.querySelector(".selected-member-list")?.scrollTop ?? 0;
  const active = document.activeElement;
  const shouldRestoreInput = active?.id === inputId;
  const selectionStart = shouldRestoreInput ? active.selectionStart : null;
  const selectionEnd = shouldRestoreInput ? active.selectionEnd : null;

  renderFn();

  const nextContainer = $(`#${containerId}`);
  const options = nextContainer?.querySelector(".member-picker-options");
  const selected = nextContainer?.querySelector(".selected-member-list");
  if (options) options.scrollTop = optionsScrollTop;
  if (selected) selected.scrollTop = selectedScrollTop;

  if (!shouldRestoreInput) return;
  const input = $(`#${inputId}`);
  if (!input) return;
  input.focus({ preventScroll: true });
  if (selectionStart !== null && selectionEnd !== null) {
    input.setSelectionRange(selectionStart, selectionEnd);
  }
}

export function setNotificationPopover(open) {
  state.notificationPopoverOpen = open;
  renderNotifications();
}

export function setUserMenu(open) {
  state.userMenuOpen = open;
  renderUserMenu();
}

export function setConfigFileCreate(open) {
  state.configFileCreateOpen = open;
  state.configFileMenuOpen = false;
  if (open) {
    state.configFileSourceType = "blank";
    state.configFileSourceId = "";
    state.configFileDraftName = "";
  }
  renderConfigRows();
  if (open) {
    window.setTimeout(() => $("#newConfigFileName")?.focus(), 0);
  }
}

export function setConfigFileMenu(open) {
  state.configFileMenuOpen = Boolean(open);
  renderConfigRows();
}

export function toggleConfigFileMenu() {
  setConfigFileMenu(!state.configFileMenuOpen);
}

export function setConfigSourceModal(sourceType = "", open = false) {
  state.configSourceModalOpen = Boolean(open);
  state.configSourceModalType = open ? sourceType : "";
  state.configFileMenuOpen = false;
  state.configFileCreateOpen = false;
  renderConfigRows();
}

export function updateConfigFileDraftName(name) {
  state.configFileDraftName = name || "";
}

export function updateConfigFileSourceType(sourceType) {
  state.configFileDraftName =
    $("#newConfigFileName")?.value || state.configFileDraftName || "";
  state.configFileSourceType = sourceType || "blank";
  state.configFileSourceId = "";
  renderConfigRows();
}

export function updateConfigFileSourceId(sourceId) {
  state.configFileSourceId = sourceId || "";
}

export async function createConfigFile(event) {
  event.preventDefault();
  const project = activeProject();
  const name = $("#newConfigFileName")?.value || "";
  const sourceType = state.configFileSourceType || "blank";
  const sourceId = state.configFileSourceId || $("#configFileSourceId")?.value || "";
  const source = configFileSource(sourceType, sourceId);
  if (!project) {
    showToast("Choose a project first");
    return;
  }
  if (!name.trim()) {
    showToast("Config name is required");
    return;
  }
  const created = await api(`/projects/${project.id}/configs`, {
    method: "POST",
    body: JSON.stringify({
      name,
      sourceType,
      sourceId,
      description: source.label,
    }),
  });
  state.activeConfigFile = created.id;
  await loadConfigsAndHistory();
  state.configFileCreateOpen = false;
  state.configFileDraftName = "";
  renderAll();
  showToast(`${created.name} created`);
}

function configFileSource(sourceType, sourceId) {
  if (sourceType === "template") {
    const template = state.templates.find((item) => item.id === sourceId);
    return {
      type: sourceType,
      id: sourceId,
      item: template,
      label: template ? `Template: ${template.name}` : "Template source",
    };
  }
  if (sourceType === "shared-config") {
    const sharedConfig = state.sharedConfigs.find(
      (item) => item.id === sourceId,
    );
    return {
      type: sourceType,
      id: sourceId,
      item: sharedConfig,
      label: sharedConfig
        ? `Shared: ${sharedConfig.name}`
        : "Shared config source",
    };
  }
  return { type: "blank", id: "", item: null, label: "Custom config file" };
}

export async function setConfigMode(mode) {
  if (!["view", "compare"].includes(mode)) return;
  state.configMode = mode;
  state.inlineEdit = null;
  if (mode === "compare") {
    await loadCompareConfigs();
  }
  renderConfigRows();
}

export async function setActiveBranch(branch) {
  const project = activeProject();
  const branches = projectBranches(project);
  const nextBranch = String(branch || "").trim().toLowerCase();
  if (!nextBranch || !branches.includes(nextBranch)) return;
  state.activeBranch = nextBranch;
  state.inlineEdit = null;
  state.activeConfigFile = "";
  await loadConfigsAndHistory();
  if (state.configMode === "compare") {
    await loadCompareConfigs();
  }
  renderAll();
}

export async function setCompareEnvironment(side, environment) {
  if (!environment || !["source", "target"].includes(side)) return;
  if (side === "source") {
    state.compareSourceEnv = environment;
  } else {
    state.compareTargetEnv = environment;
  }
  await loadCompareConfigs();
  renderConfigRows();
}

export function setConfigCreateDrawer(open) {
  state.configCreateDrawerOpen = open;
  if (open) {
    state.newConfigValueType = "string";
    renderConfigCreateDrawer();
    window.setTimeout(() => $("#newConfigKey")?.focus(), 0);
    return;
  }
  $("#configCreateForm")?.reset();
  state.newConfigValueType = "string";
  renderConfigCreateDrawer();
}

export function updateNewConfigValueType(valueType) {
  state.newConfigValueType = valueType || "string";
  renderConfigCreateDrawer();
}

export async function createConfigKey(event) {
  event.preventDefault();
  const project = activeProject();
  if (!project) {
    showToast("Choose a project first");
    return;
  }

  const key = $("#newConfigKey").value.trim();
  const reason = $("#newConfigReason").value.trim();
  if (!key) {
    showToast("Config key is required");
    $("#newConfigKey").focus();
    return;
  }
  if (!reason) {
    showToast("Change Reason is required");
    $("#newConfigReason").focus();
    return;
  }

  const requests = Array.from(
    document.querySelectorAll("[data-new-config-value]"),
  )
    .map((input) => ({
      environment: input.dataset.newConfigValue,
      branch: state.activeBranch,
      value: input.value,
    }))
    .filter(({ environment, value }) => environment && value.trim() !== "");

  if (!requests.length) {
    showToast("Enter at least one environment value");
    return;
  }
  if (!state.activeConfigFile) {
    showToast("Add or select a config first");
    return;
  }

  const bodyBase = {
    key,
    configId: state.activeConfigFile,
    valueType: $("#newConfigValueType").value,
    isSensitive: $("#newConfigSensitive").checked,
    changeReason: reason,
  };

  const directRequests = requests.filter((request) =>
    canDirectWriteEnvironment(project, request.environment),
  );
  const stagedRequests = requests.filter((request) =>
    shouldStageReviewChange(project, request.environment),
  );
  const blockedRequests = requests.filter(
    (request) =>
      !canDirectWriteEnvironment(project, request.environment) &&
      !shouldStageReviewChange(project, request.environment),
  );

  if (blockedRequests.length) {
    showToast("You cannot change one or more selected environments");
    return;
  }

  for (const request of directRequests) {
    await api(`/projects/${project.id}/configs`, {
      method: "POST",
      body: JSON.stringify({ ...bodyBase, ...request }),
    });
  }

  if (directRequests.length) {
    await loadConfigsAndHistory();
    if (!stagedRequests.length) {
      acceptCurrentConfigsAsBaseline();
    }
  }
  stagedRequests.forEach((request) =>
    stageLocalConfigCreate(project, request, bodyBase),
  );

  if (state.configMode === "compare") {
    await loadCompareConfigs();
  }
  setConfigCreateDrawer(false);
  renderAll();
  const stagedCopy = stagedRequests.length
    ? ` (${stagedRequests.length} pending review)`
    : "";
  showToast(
    `${key} created in ${requests.length} environment${requests.length === 1 ? "" : "s"}${stagedCopy}`,
  );
}

export async function openGroupPanel() {
  state.userMenuOpen = false;
  state.groupModalOpen = true;
  clearGroupMemberPicker();
  state.groupCreateOpen = false;
  state.groupLoading = true;
  state.groupError = "";
  renderUserMenu();
  renderGroupModal();
  try {
    await Promise.all([reloadGroups(), reloadUsers({ silent: true })]);
    if (state.activeGroupId) {
      await reloadGroupDetail(state.activeGroupId);
    }
  } catch (error) {
    state.groupError = error.message;
  } finally {
    state.groupLoading = false;
    renderGroupModal();
  }
}

export function setGroupModal(open) {
  state.groupModalOpen = open;
  clearGroupMemberPicker();
  state.groupCreateOpen = false;
  clearGroupCreatePicker();
  state.groupError = "";
  renderGroupModal();
}

export async function selectGroup(groupId) {
  state.activeGroupId = groupId;
  state.groupLoading = true;
  clearGroupMemberPicker();
  state.groupCreateOpen = false;
  renderGroupModal();
  try {
    await reloadGroupDetail(groupId);
    state.groupError = "";
  } catch (error) {
    state.groupError = error.message;
  } finally {
    state.groupLoading = false;
    renderGroupModal();
  }
}

export function toggleGroupCreate(open = !state.groupCreateOpen) {
  state.groupCreateOpen = open;
  clearGroupMemberPicker();
  if (open) {
    clearGroupCreatePicker();
  }
  renderGroupModal();
  if (open) {
    window.setTimeout(() => $("#groupName")?.focus(), 0);
  }
}

export function setGroupDetailTab(tab) {
  if (!["members", "projects"].includes(tab)) return;
  state.groupDetailTab = tab;
  clearGroupMemberPicker();
  renderGroupModal();
}

export function updateGroupCreateMemberSearch(value) {
  state.groupCreateMemberSearch = value;
  renderPreservingMemberPicker(
    "groupCreateMemberPicker",
    "groupCreateMemberSearch",
    renderGroupCreateMemberPicker,
  );
}

export function toggleGroupCreateMemberSelection(userId, selected) {
  if (!userId) return;
  if (selected) {
    state.groupCreateMemberSelection.add(userId);
  } else {
    state.groupCreateMemberSelection.delete(userId);
  }
  renderPreservingMemberPicker(
    "groupCreateMemberPicker",
    "groupCreateMemberSearch",
    renderGroupCreateMemberPicker,
  );
}

export function removeSelectedCreateMember(userId) {
  state.groupCreateMemberSelection.delete(userId);
  renderPreservingMemberPicker(
    "groupCreateMemberPicker",
    "groupCreateMemberSearch",
    renderGroupCreateMemberPicker,
  );
}

export async function createGroup(event) {
  event.preventDefault();
  if (!canCreateGroup()) {
    showToast("Only system admins can create groups");
    return;
  }
  const selectedMembers = Array.from(state.groupCreateMemberSelection);
  const created = await api("/groups", {
    method: "POST",
    body: JSON.stringify({
      name: $("#groupName").value,
      memberIds: selectedMembers,
    }),
  });
  const groupId = created.id || created.groupId;
  for (const userId of selectedMembers) {
    try {
      await api(`/groups/${encodeURIComponent(groupId)}/members`, {
        method: "POST",
        body: JSON.stringify({ userId }),
      });
    } catch (error) {
      // Some backends may already apply memberIds during group creation.
    }
  }
  await reloadGroups();
  await reloadGroupDetail(groupId);
  state.groupCreateOpen = false;
  clearGroupCreatePicker();
  $("#groupForm").reset();
  renderGroupModal();
  showToast(`${created.name || created.groupName} created`);
}

export function toggleGroupMemberPicker(open = !state.groupMemberPickerOpen) {
  state.groupMemberPickerOpen = open;
  state.groupMemberSearch = "";
  state.groupMemberSelection = new Set();
  state.groupRoleMenuUserId = "";
  renderGroupModal();
}

export function toggleGroupRoleMenu(userId) {
  state.groupRoleMenuUserId =
    state.groupRoleMenuUserId === userId ? "" : userId;
  renderGroupModal();
}

export function updateGroupMemberSearch(value) {
  state.groupMemberSearch = value;
  renderPreservingMemberPicker(
    "groupMemberTools",
    "groupMemberSearch",
    renderGroupMemberPicker,
  );
}

export function toggleGroupMemberSelection(userId, selected) {
  if (!userId) return;
  if (selected) {
    state.groupMemberSelection.add(userId);
  } else {
    state.groupMemberSelection.delete(userId);
  }
  renderPreservingMemberPicker(
    "groupMemberTools",
    "groupMemberSearch",
    renderGroupMemberPicker,
  );
}

export function removeSelectedGroupMember(userId) {
  state.groupMemberSelection.delete(userId);
  renderPreservingMemberPicker(
    "groupMemberTools",
    "groupMemberSearch",
    renderGroupMemberPicker,
  );
}

export async function addGroupMember() {
  if (!canEditGroupMembers()) {
    showToast("Only admins can edit group members");
    return;
  }
  const groupId = state.activeGroupId;
  const userIds = Array.from(state.groupMemberSelection);
  if (!groupId || userIds.length === 0) return;
  for (const userId of userIds) {
    await api(`/groups/${encodeURIComponent(groupId)}/members`, {
      method: "POST",
      body: JSON.stringify({ userId }),
    });
  }
  await reloadGroups();
  await reloadGroupDetail(groupId);
  clearGroupMemberPicker();
  renderGroupModal();
  showToast(`${userIds.length} member${userIds.length === 1 ? "" : "s"} added`);
}

export async function removeGroupMember(userId) {
  if (!canEditGroupMembers()) {
    showToast("Only admins can edit group members");
    return;
  }
  const groupId = state.activeGroupId;
  if (!groupId || !userId) return;
  await api(
    `/groups/${encodeURIComponent(groupId)}/members/${encodeURIComponent(userId)}`,
    {
      method: "DELETE",
    },
  );
  await reloadGroups();
  await reloadGroupDetail(groupId);
  renderGroupModal();
  showToast("Member removed");
}

export async function updateGroupMemberRole(userId, groupRole) {
  if (!canEditGroupMembers()) {
    showToast("Only admins can edit group members");
    return;
  }
  const groupId = state.activeGroupId;
  if (!groupId || !userId) return;
  await api(`/groups/${encodeURIComponent(groupId)}/members`, {
    method: "POST",
    body: JSON.stringify({ userId, groupRole }),
  });
  await reloadGroups();
  await reloadGroupDetail(groupId);
  state.groupRoleMenuUserId = "";
  renderGroupModal();
  showToast("Member role updated");
}

export function setProjectModal(open) {
  state.projectModalOpen = open;
  $("#projectModal").classList.toggle("hidden", !open);
  if (open) {
    restoreProjectDraft();
    renderProjectFormOptions();
    window.setTimeout(() => $("#projectName").focus(), 0);
  } else {
    $("#projectForm").reset();
    $("#projectEnvironments").value = "dev, staging, prod";
    if ($("#projectBranches")) $("#projectBranches").value = "default";
    state.projectDraft = null;
    renderProjectFormOptions();
  }
}

export function openConfigSourcePicker(sourceType) {
  if (!sourceType) {
    setConfigSourceModal("", false);
    return;
  }
  setConfigSourceModal(sourceType, true);
}

export function cancelConfigSourcePicker() {
  setConfigSourceModal("", false);
}

export function chooseConfigSource(sourceType, sourceId) {
  state.configFileSourceType = sourceType;
  state.configFileSourceId = sourceId || "";
  setConfigFileCreate(true);
}

export async function importConfigSource(sourceType, sourceId) {
  const project = activeProject();
  if (!project) {
    showToast("Choose a project first");
    return;
  }
  const source = configFileSource(sourceType, sourceId);
  if (!source.id) {
    showToast("Choose a source first");
    return;
  }
  const created = await api(`/projects/${project.id}/configs`, {
    method: "POST",
    body: JSON.stringify({
      name: configFileNameForSource(sourceType, source.item),
      sourceType,
      sourceId,
      description: source.label,
    }),
  });
  await loadConfigsAndHistory();
  state.activeConfigFile = created.id;
  state.configSourceModalOpen = false;
  state.configSourceModalType = "";
  renderAll();
  showToast(`${created.name} imported`);
}

function configFileNameForSource(sourceType, item) {
  const id = item?.id || sourceType;
  const cleaned = id
    .replace(/^(global|group|project)-/i, "")
    .replace(/-template$/i, "")
    .replace(/[^a-z0-9._-]+/gi, "-")
    .replace(/^-+|-+$/g, "")
    .toLowerCase();
  return `${cleaned || "config"}.yaml`;
}

function readProjectDraft() {
  return {
    name: $("#projectName").value,
    repoUrl: $("#projectRepo").value,
    environments: $("#projectEnvironments").value,
    description: $("#projectDescription").value,
    branches: $("#projectBranches")?.value || "default",
    groupId: $("#projectGroup")?.value || "",
  };
}

function restoreProjectDraft() {
  const draft = state.projectDraft;
  if (!draft) return;
  $("#projectName").value = draft.name || "";
  $("#projectRepo").value = draft.repoUrl || "";
  $("#projectEnvironments").value = draft.environments || "dev, staging, prod";
  $("#projectDescription").value = draft.description || "";
  if ($("#projectBranches")) $("#projectBranches").value = draft.branches || "default";
  const groupSelect = $("#projectGroup");
  if (groupSelect) groupSelect.value = draft.groupId || groupSelect.value || "";
}

function sharedConfigById(id) {
  return state.sharedConfigs.find((config) => config.id === id);
}

function sharedConfigEntriesForTextarea(item) {
  return JSON.stringify(item?.entries || [], null, 2);
}

export function openSharedConfigEdit(id) {
  const item = sharedConfigById(id);
  if (!item) return;
  state.activeSharedConfigId = id;
  state.sharedConfigEditModalOpen = true;
  renderSharedConfigEditModal();
  $("#sharedConfigEditName").value = item.name || "";
  $("#sharedConfigEditDescription").value = item.description || "";
  $("#sharedConfigEditFormat").value = item.format || "yaml";
  $("#sharedConfigEditEntries").value = sharedConfigEntriesForTextarea(item);
  $("#sharedConfigChangeReason").value = "";
  window.setTimeout(() => $("#sharedConfigEditName")?.focus(), 0);
}

export function setSharedConfigEditModal(open) {
  state.sharedConfigEditModalOpen = open;
  if (!open) {
    state.activeSharedConfigId = "";
    $("#sharedConfigEditForm")?.reset();
  }
  renderSharedConfigEditModal();
}

function readSharedConfigEditEntries() {
  try {
    const parsed = JSON.parse($("#sharedConfigEditEntries").value || "[]");
    return Array.isArray(parsed) ? parsed : [];
  } catch (error) {
    throw new Error("Entries JSON is invalid");
  }
}

export async function updateSharedConfig(event) {
  event.preventDefault();
  const id = state.activeSharedConfigId;
  const item = sharedConfigById(id);
  if (!item) return;
  const changeReason = $("#sharedConfigChangeReason").value.trim();
  if (!changeReason) {
    showToast("Change Reason is required");
    return;
  }
  const affected = item.inheritedBy || item.affectedProjects?.length || 0;
  const prod = item.prodEnvironmentCount || 0;
  const confirmed = window.confirm(
    `This change will affect ${affected} projects and ${prod} production environments. Apply it now?`,
  );
  if (!confirmed) return;
  const updated = await api(`/shared-configs/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: JSON.stringify({
      name: $("#sharedConfigEditName").value,
      description: $("#sharedConfigEditDescription").value,
      format: $("#sharedConfigEditFormat").value,
      entries: readSharedConfigEditEntries(),
      changeReason,
    }),
  });
  await Promise.all([reloadSharedConfigs(), reloadNotifications()]);
  setSharedConfigEditModal(false);
  renderAll();
  showToast(`${updated.name} updated`);
}

function populateSharedConfigScopeControls() {
  const scopeSelect = $("#sharedConfigScope");
  const groupSelect = $("#sharedConfigGroup");
  const groupField = $("#sharedConfigGroupField");
  if (!scopeSelect || !groupSelect || !groupField) return;

  const manageableGroups = manageableSharedConfigGroups();
  const isSystem = state.user?.role === "system_admin";
  scopeSelect.innerHTML = `${isSystem ? '<option value="global">Global</option>' : ""}<option value="group">Group</option>`;
  if (!isSystem) {
    scopeSelect.value = "group";
  } else if (!scopeSelect.value) {
    scopeSelect.value = "global";
  }
  groupSelect.innerHTML = manageableGroups
    .map((group) => `<option value="${group.id}">${group.name || group.id}</option>`)
    .join("");
  groupField.classList.toggle("hidden", scopeSelect.value !== "group");
}

export function setSharedConfigCreateModal(open) {
  state.sharedConfigCreateModalOpen = open;
  renderSharedConfigCreateModal();
  if (open) {
    populateSharedConfigScopeControls();
    const textarea = $("#sharedConfigEntries");
    if (textarea && !textarea.value.trim()) {
      textarea.value = JSON.stringify(
        [
          {
            key: "logging.level.root",
            value: "INFO",
            valueType: "string",
            environment: "prod",
            isSensitive: false,
          },
        ],
        null,
        2,
      );
    }
    window.setTimeout(() => $("#sharedConfigName")?.focus(), 0);
  } else {
    $("#sharedConfigCreateForm")?.reset();
  }
}

function readSharedConfigEntries() {
  try {
    const parsed = JSON.parse($("#sharedConfigEntries").value || "[]");
    return Array.isArray(parsed) ? parsed : [];
  } catch (error) {
    throw new Error("Entries JSON is invalid");
  }
}

export async function createSharedConfig(event) {
  event.preventDefault();
  const created = await api("/shared-configs", {
    method: "POST",
    body: JSON.stringify({
      name: $("#sharedConfigName").value,
      description: $("#sharedConfigDescription").value,
      scope: $("#sharedConfigScope")?.value || "global",
      scopeId: $("#sharedConfigScope")?.value === "group" ? $("#sharedConfigGroup")?.value || "" : "",
      format: $("#sharedConfigFormat").value,
      entries: readSharedConfigEntries(),
    }),
  });
  await reloadSharedConfigs();
  setSharedConfigCreateModal(false);
  renderAll();
  showToast(`${created.name} created`);
}

export async function deleteSharedConfig(id) {
  const item = state.sharedConfigs.find((config) => config.id === id);
  if (!item) return;
  if (!window.confirm(`Delete shared config ${item.name}?`)) return;
  await api(`/shared-configs/${encodeURIComponent(id)}`, { method: "DELETE" });
  await reloadSharedConfigs();
  renderAll();
  showToast(`${item.name} deleted`);
}

export async function submitSharedConfigUpdate(id) {
  const item = state.sharedConfigs.find((config) => config.id === id);
  if (!item) return;
  const reason = window.prompt(
    `Submit update request for ${item.name}`,
    "Update shared config",
  );
  if (!reason || !reason.trim()) return;
  await api(`/shared-configs/${encodeURIComponent(id)}/submit-update`, {
    method: "POST",
    body: JSON.stringify({
      name: item.name,
      description: item.description,
      format: item.format,
      entries: item.entries,
      reason,
    }),
  });
  showToast("Shared config update submitted");
}

export function setTemplateCreateModal(open) {
  state.templateCreateModalOpen = open;
  renderTemplateCreateModal();
  if (open) {
    window.setTimeout(() => $("#templateName").focus(), 0);
  } else {
    $("#templateCreateForm").reset();
    $("#templateFormat").value = "yaml";
  }
}


export function setExportModal(open) {
  state.exportModalOpen = open;
  renderExportModal();
}

export function setImportModal(open) {
  state.importModalOpen = open;
  $("#importModal")?.classList.toggle("hidden", !open);
}

export function setImportPreviewModal(open) {
  state.importPreviewOpen = open;
  renderImportPreview();
}

export function setHistoryModal(open) {
  state.historyModalOpen = open;
  if (!open) {
    state.historyLoading = false;
  }
  renderVersionHistory();
}

export function setReviewModal(open) {
  state.reviewModalOpen = open;
  if (open) {
    const reason = $("#reviewReason");
    if (reason) {
      reason.value = "";
      reason.dataset.touched = "";
    }
  }
  renderReviewModal();
  if (open) {
    window.setTimeout(() => $("#reviewReason").focus(), 0);
  }
}

export function setRequestDetailModal(open, requestId = state.activeRequestId) {
  state.requestDetailOpen = open;
  state.activeRequestId = open ? requestId : "";
  renderRequestDetailModal();
}


export function setTemplateModal(open, templateId = "") {
  state.templateModalOpen = open;
  if (open) {
    const template = state.templates.find((item) => item.id === templateId);
    state.activeTemplateId = templateId;
    state.templateValues = Object.fromEntries(
      (template?.variables || []).map((variable) => [
        variable.name,
        variable.defaultValue || "",
      ]),
    );
    state.templateApplyFormat =
      state.templateApplyFormat || template?.format || "yaml";
  }
  renderTemplateModal();
}

export function updateTemplateValue(name, value) {
  state.templateValues[name] = value;
  renderTemplateModal();
}

export async function createTemplate(event) {
  event.preventDefault();
  const created = await api("/templates", {
    method: "POST",
    body: JSON.stringify({
      name: $("#templateName").value,
      description: $("#templateDescription").value,
      format: $("#templateFormat").value,
      body: $("#templateBody").value,
    }),
  });

  await reloadTemplates();
  setTemplateCreateModal(false);
  renderAll();
  showToast(`${created.name} created`);
}

export async function applyTemplate() {
  const template = state.templates.find(
    (item) => item.id === state.activeTemplateId,
  );
  const project = activeProject();
  if (!template || !project) {
    showToast("Choose a template and project first");
    return;
  }
  if (!state.activeConfigFile) {
    showToast("Add or select a config first");
    return;
  }
  for (const variable of template.variables || []) {
    if (
      variable.required &&
      String(state.templateValues[variable.name] || "").trim() === ""
    ) {
      showToast(`${variable.name} is required`);
      return;
    }
  }
  const sourceFormat = template.format || "yaml";
  const outputFormat =
    $("#templateApplyFormat")?.value ||
    state.templateApplyFormat ||
    sourceFormat;
  state.templateApplyFormat = outputFormat;
  const content = renderTemplateContent(template);
  const result = await api(`/projects/${project.id}/configs/extract`, {
    method: "POST",
    body: JSON.stringify({
      environment: state.activeEnvironment,
      branch: state.activeBranch,
      configId: state.activeConfigFile,
      format: sourceFormat,
      content,
    }),
  });
  const outputContent =
    outputFormat === sourceFormat
      ? content
      : serializeConfigs(result.entries || [], outputFormat);
  state.importPreview = {
    ...result,
    fileName: template.name,
    content: outputContent,
    format: outputFormat,
    projectId: project.id,
    environment: state.activeEnvironment,
    branch: state.activeBranch,
    configId: state.activeConfigFile,
  };
  state.templateModalOpen = false;
  state.importPreviewOpen = true;
  renderAll();
  showToast(`Extracted ${result.entryCount} config keys`);
}

export async function createProject(event) {
  event.preventDefault();
  const created = await api("/projects", {
    method: "POST",
    body: JSON.stringify({
      name: $("#projectName").value,
      repoUrl: $("#projectRepo").value,
      groupId: $("#projectGroup")?.value || "",
      environments: parseEnvironmentInput($("#projectEnvironments").value),
      branches: parseEnvironmentInput($("#projectBranches")?.value || "default"),
      description: $("#projectDescription").value,
    }),
  });

  await Promise.all([reloadProjects(), reloadGroups({ silent: true })]);
  state.activeProjectId = created.id;
  const project = activeProject();
  if (project) {
    state.activeEnvironment = project.environments.includes("prod")
      ? "prod"
      : project.environments[0];
    state.activeBranch = defaultBranchForProject(project);
  }
  await loadConfigsAndHistory();
  setProjectModal(false);
  switchView("projects");
  renderAll();
  showToast(`${created.name} created`);
}


export async function openVersionHistory() {
  const project = activeProject();
  if (!project) {
    showToast("Choose a project first");
    return;
  }
  state.historyModalOpen = true;
  state.historyLoading = true;
  renderVersionHistory();
  try {
    await loadConfigHistory();
  } finally {
    state.historyLoading = false;
    renderVersionHistory();
  }
}


function stageRollbackRevision(project, revision) {
  const revisionEntries = revision?.entries || [];
  for (const entry of revisionEntries) {
    const existing = state.configs.find(
      (config) =>
        config.key === entry.key &&
        (config.configId || "") === (entry.configId || ""),
    );
    if (existing) {
      state.configs = state.configs.map((config) =>
        config.id === existing.id
          ? {
              ...config,
              configId: entry.configId || config.configId,
              key: entry.key,
              value: entry.value,
              valueType: entry.valueType || config.valueType || "string",
              isSensitive: Boolean(entry.isSensitive),
            }
          : config,
      );
      continue;
    }
    const id = `draft-rollback-${project.id}-${state.activeBranch}-${state.activeEnvironment}-${entry.key}`.replace(/[^a-z0-9_-]+/gi, "-");
    state.configs = [
      ...state.configs,
      {
        id,
        projectId: project.id,
        environment: state.activeEnvironment,
        branch: state.activeBranch,
        configId: entry.configId || state.activeConfigFile,
        key: entry.key,
        value: entry.value,
        valueType: entry.valueType || "string",
        isSensitive: Boolean(entry.isSensitive),
      },
    ];
  }
  syncPendingReviewChanges();
}

export async function rollbackConfigRevision(revisionId) {
  const project = activeProject();
  const revision = state.configHistory.find((item) => item.id === revisionId);
  if (!project || !revision) {
    showToast("Choose a config revision to restore");
    return;
  }

  if (shouldStageReviewChange(project, state.activeEnvironment)) {
    stageRollbackRevision(project, revision);
    state.historyModalOpen = false;
    renderAll();
    setReviewModal(true);
    showToast("Rollback staged for review");
    return;
  }

  const version = String(revision.id || "").replace(/^rev-/, "").slice(0, 7);
  const confirmed = window.confirm(
    `Rollback ${project.name} ${state.activeEnvironment} config to version ${version}?`,
  );
  if (!confirmed) return;

  await api(`/projects/${project.id}/config-history/rollback`, {
    method: "POST",
    body: JSON.stringify({
      environment: state.activeEnvironment,
      branch: state.activeBranch,
      revisionId: revision.id,
      changeReason: `rollback project config to ${version}`,
    }),
  });

  await loadConfigsAndHistory();
  renderAll();
  showToast(`${project.name} ${state.activeEnvironment} config rolled back`);
}

export function startInlineEdit(configId, field) {
  const config = state.configs.find((entry) => entry.id === configId);
  if (!config) return;

  if (isSharedConfigEntry(config) && field === "key") {
    showToast("Shared config keys can be overridden by value only");
    return;
  }

  if (field === "value" && config.isSensitive) {
    const revealKey = `${config.projectId}:${config.environment}:${config.key}`;
    if (!state.revealedKeys.has(revealKey)) {
      showToast("Reveal this sensitive value before editing it");
      return;
    }
  }

  const previousConfigId = state.inlineEdit?.configId;
  state.inlineEdit = {
    configId,
    field,
    value: field === "key" ? config.key : config.value,
  };
  if (previousConfigId && previousConfigId !== configId) {
    renderConfigRow(previousConfigId);
  }
  renderConfigRow(configId);
  window.setTimeout(() => {
    const input = document.querySelector(
      `[data-inline-input][data-config-id="${CSS.escape(configId)}"][data-field="${field}"]`,
    );
    input?.focus({ preventScroll: true });
  }, 0);
}

export function cancelInlineEdit() {
  const configId = state.inlineEdit?.configId;
  state.inlineEdit = null;
  if (configId) {
    renderConfigRow(configId);
    return;
  }
  renderConfigRows(false);
}

export async function commitInlineEdit(input) {
  const edit = state.inlineEdit;
  if (!edit || state.inlineSaving) return;
  const config = state.configs.find((entry) => entry.id === edit.configId);
  if (!config) return;

  const rawValue = input.value;
  const nextValue = edit.field === "key" ? rawValue.trim() : rawValue;
  if (edit.field === "key" && !nextValue) {
    showToast("Config key is required");
    input.focus();
    return;
  }
  if (nextValue === edit.value) {
    state.inlineEdit = null;
    renderConfigRow(edit.configId);
    return;
  }

  const project = activeProject();
  if (shouldStageReviewChange(project, config.environment)) {
    stageLocalConfigChange(config.id, edit.field, nextValue);
    state.inlineEdit = null;
    renderAll();
    showToast(`${config.key} staged for review`);
    return;
  }

  state.inlineSaving = true;
  try {
    let updated;
    if (config.inherited) {
      updated = await api(`/projects/${config.projectId}/configs`, {
        method: "POST",
        body: JSON.stringify({
          environment: config.environment,
          branch: config.branch || state.activeBranch,
          configId: config.configId,
          key: config.key,
          value: nextValue,
          valueType: config.valueType || "string",
          isSensitive: Boolean(config.isSensitive),
          changeReason: "local override for shared config",
        }),
      });
    } else {
      const body =
        edit.field === "key"
          ? { key: nextValue, changeReason: "inline key edit from frontend" }
          : { value: nextValue, changeReason: "inline value edit from frontend" };
      updated = await api(
        `/projects/${config.projectId}/configs/${config.id}`,
        {
          method: "PUT",
          body: JSON.stringify(body),
        },
      );
    }
    state.inlineEdit = null;
    await loadConfigsAndHistory();
    acceptCurrentConfigsAsBaseline();
    renderAll();
    showToast(`${updated.key} updated`);
  } finally {
    state.inlineSaving = false;
  }
}

function acceptCurrentConfigsAsBaseline() {
  state.configBaseline = new Map(
    state.configs.map((config) => [
      config.id,
      {
        id: config.id,
        key: config.key,
        value: config.value,
        environment: config.environment,
        branch: config.branch || state.activeBranch,
      },
    ]),
  );
  state.pendingReviewChanges = [];
}

function syncPendingReviewChanges() {
  state.pendingReviewChanges = state.configs
    .filter((config) => configHasNetChange(config))
    .map((config) => ({
      configId: config.id,
      key: config.key,
      environment: config.environment,
      branch: config.branch || state.activeBranch,
    }));
}

function configHasNetChange(config) {
  const baseline = state.configBaseline.get(config.id);
  if (!baseline) return true;
  return (
    baseline.key !== config.key ||
    baseline.value !== config.value ||
    (baseline.branch || state.activeBranch) !== (config.branch || state.activeBranch)
  );
}

export async function hasReviewRequest(config) {
  const requests = await api(
    `/projects/${config.projectId}/review-requests?env=prod&key=${encodeURIComponent(
      config.key,
    )}&branch=${encodeURIComponent(config.branch || state.activeBranch)}&status=pending`,
  );
  return requests.length > 0;
}

export async function extractConfigFile() {
  const file = $("#configFile").files[0];
  const project = activeProject();
  if (!file || !project) {
    showToast("Choose a config file first");
    return;
  }
  if (!state.activeConfigFile) {
    showToast("Add or select a config first");
    return;
  }

  const format = $("#configFormat").value;
  const content = await file.text();
  const result = await api(`/projects/${project.id}/configs/extract`, {
    method: "POST",
    body: JSON.stringify({
      environment: state.activeEnvironment,
      branch: state.activeBranch,
      configId: state.activeConfigFile,
      format,
      content,
    }),
  });

  state.importPreview = {
    ...result,
    fileName: file.name,
    content,
    format,
    projectId: project.id,
    environment: state.activeEnvironment,
    branch: state.activeBranch,
    configId: state.activeConfigFile,
  };
  state.importPreviewOpen = true;
  setImportModal(false);
  renderImportPreview();
  showToast(`Extracted ${result.entryCount} config keys`);
}

export async function applyImportPreview() {
  const preview = state.importPreview;
  if (!preview) {
    showToast("Extract a config file first");
    return;
  }

  state.importApplying = true;
  renderImportPreview();
  try {
    const result = await api(`/projects/${preview.projectId}/configs/import`, {
      method: "POST",
      body: JSON.stringify({
        environment: preview.environment,
        branch: preview.branch || state.activeBranch,
        configId: preview.configId || state.activeConfigFile,
        format: preview.format,
        content: preview.content,
        changeReason: `import ${preview.fileName}`,
      }),
    });

    await loadConfigsAndHistory();
    acceptCurrentConfigsAsBaseline();
    state.importPreviewOpen = false;
    state.importPreview = null;
    showToast(
      `Imported ${result.imported}: ${result.created} created, ${result.updated} updated`,
    );
  } finally {
    state.importApplying = false;
    renderAll();
  }
}

export function openExportConfig() {
  const project = activeProject();
  if (!project) {
    showToast("Choose a project first");
    return;
  }
  setExportModal(true);
}

export async function exportCurrentConfig() {
  const project = activeProject();
  if (!project) {
    showToast("Choose a project first");
    return;
  }

  const format = $("#exportFormat").value || state.exportFormat || "yaml";
  state.exportFormat = format;

  let payload;
  try {
    payload = await api(
      `/projects/${project.id}/configs?env=${encodeURIComponent(state.activeEnvironment)}&branch=${encodeURIComponent(state.activeBranch)}&revealSensitive=true`,
    );
  } catch (error) {
    showToast("Export denied: your role cannot reveal sensitive values");
    return;
  }

  const configs = payload.entries.map((entry) => ({
    ...entry,
    updated: entry.updatedBy,
  }));
  const content = serializeConfigs(configs, format);
  const extension =
    format === "properties"
      ? "properties"
      : format === "json"
        ? "json"
        : "yaml";
  downloadFile(
    `${project.name}-${state.activeBranch}-${state.activeEnvironment}.${extension}`,
    content,
    format === "json" ? "application/json" : "text/plain",
  );
  state.exportModalOpen = false;
  renderExportModal();
  showToast("Config file exported");
}

function serializeConfigs(configs, format) {
  const entries = [...configs].sort((a, b) => a.key.localeCompare(b.key));
  if (format === "json") {
    return `${JSON.stringify(Object.fromEntries(entries.map((entry) => [entry.key, entry.value])), null, 2)}\n`;
  }
  if (format === "properties") {
    return (
      entries.map((entry) => `${entry.key}=${entry.value}`).join("\n") + "\n"
    );
  }
  return (
    entries
      .map((entry) => `${entry.key}: ${JSON.stringify(entry.value)}`)
      .join("\n") + "\n"
  );
}

function downloadFile(filename, content, type) {
  const blob = new Blob([content], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

export async function createReviewRequest() {
  if (state.pendingReviewChanges.length === 0) {
    showToast("No config changes to review");
    return;
  }
  setReviewModal(true);
}


function proposedReviewChanges() {
  return state.pendingReviewChanges
    .map((change) => {
      const config = state.configs.find((entry) => entry.id === change.configId);
      if (!config) return null;
      const id = String(config.id || "");
      const entryId = id.startsWith("draft-") || id.startsWith("inherited-")
        ? ""
        : config.id;
      const baseline = state.configBaseline.get(change.configId);
      return {
        configEntryId: entryId,
        configId: config.configId,
        key: config.key,
        oldValue: baseline?.value ?? "",
        value: config.value,
        valueType: config.valueType || "string",
        isSensitive: Boolean(config.isSensitive),
        environment: config.environment || state.activeEnvironment,
        branch: config.branch || state.activeBranch,
      };
    })
    .filter(Boolean);
}
export async function submitReviewChanges() {
  const project = activeProject();
  if (!project) return;
  const pending = state.pendingReviewChanges[0];
  const reason = $("#reviewReason").value.trim();
  if (!reason) {
    showToast("Review reason is required");
    $("#reviewReason").focus();
    return;
  }

  const proposedChanges = proposedReviewChanges();
  await api("/review-requests", {
    method: "POST",
    body: JSON.stringify({
      projectId: project.id,
      environment: state.activeEnvironment,
      branch: state.activeBranch,
      configKey: pending.key,
      reason,
      proposedChanges,
    }),
  });

  state.requests = await api("/review-requests");
  state.pendingReviewChanges = [];
  state.reviewModalOpen = false;
  await loadConfigsAndHistory();
  renderAll();
  showToast("Review request created");
}

export async function handleReviewDecision(id, action) {
  await api(`/review-requests/${id}/${action}`, {
    method: "PUT",
    body: JSON.stringify({ comment: `${action} from frontend` }),
  });
  state.requests = await api("/review-requests");
  state.requestDetailOpen = false;
  state.activeRequestId = "";
  await loadConfigsAndHistory();
  renderAll();
  showToast(`Review request ${action}d`);
}

export async function toggleSensitiveReveal(revealKey) {
  if (state.revealedKeys.has(revealKey)) {
    state.revealedKeys.delete(revealKey);
  } else {
    state.revealedKeys.add(revealKey);
  }

  await loadConfigsAndHistory(state.revealedKeys.size > 0);
  renderConfigRows();
}
