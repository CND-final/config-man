import { api } from "./api.js";
import { loadCustomConfigFiles, newCustomConfigFile, saveCustomConfigFiles } from "./configFiles.js";
import { $, $all, showToast } from "./dom.js";
import {
  loadCompareConfigs,
  loadConfigHistory,
  loadConfigsAndHistory,
  reloadGroupDetail,
  reloadGroups,
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
  renderImportPreview,
  renderGroupCreateMemberPicker,
  renderGroupMemberPicker,
  renderGroupModal,
  renderProjectTemplateOptions,
  renderRequests,
  renderReviewModal,
  renderReviewDock,
  renderTemplateCreateModal,
  renderSharedConfigCreateModal,
  renderSharedConfigEditModal,
  renderTemplateModal,
  renderUserMenu,
  renderVersionHistory,
} from "./render.js";
import { activeProject, state } from "./state.js";
import { parseEnvironmentInput } from "./utils.js";

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

export function setUserMenu(open) {
  state.userMenuOpen = open;
  renderUserMenu();
}

export function setConfigFileCreate(open) {
  state.configFileCreateOpen = open;
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
  const name = $("#newConfigFileName")?.value || "";
  const sourceType = state.configFileSourceType || "blank";
  const sourceId = state.configFileSourceId || $("#configFileSourceId")?.value || "";
  const source = configFileSource(sourceType, sourceId);
  const file = newCustomConfigFile(name, source);
  if (!file.name) {
    showToast("Config file name is required");
    return;
  }
  const sourceType = state.configFileSourceType || "blank";
  const sourceId = state.configFileSourceId || $("#configFileSourceId")?.value || "";
  const created = await api(`/projects/${project.id}/config-files`, {
    method: "POST",
    body: JSON.stringify({
      name,
      sourceType,
      sourceId,
      description: configFileSource(sourceType, sourceId).label,
    }),
  });
  await loadConfigsAndHistory();
  state.activeConfigFile = created.id;
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
      label: sharedConfig
        ? `Shared: ${sharedConfig.name}`
        : "Shared config source",
    };
  }
  return { type: "blank", id: "", label: "Custom config file" };
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
      value: input.value,
    }))
    .filter(({ environment, value }) => environment && value.trim() !== "");

  if (!requests.length) {
    showToast("Enter at least one environment value");
    return;
  }

  const bodyBase = {
    key,
    configFileId: state.activeConfigFile,
    valueType: $("#newConfigValueType").value,
    isSensitive: $("#newConfigSensitive").checked,
    changeReason: reason,
  };

  for (const request of requests) {
    await api(`/projects/${project.id}/configs`, {
      method: "POST",
      body: JSON.stringify({ ...bodyBase, ...request }),
    });
  }

  await loadConfigsAndHistory();
  if (state.configMode === "compare") {
    await loadCompareConfigs();
  }
  setConfigCreateDrawer(false);
  renderAll();
  showToast(
    `${key} created in ${requests.length} environment${requests.length === 1 ? "" : "s"}`,
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
    renderProjectTemplateOptions();
    window.setTimeout(() => $("#projectName").focus(), 0);
  } else {
    $("#projectForm").reset();
    $("#projectEnvironments").value = "dev, staging, prod";
    state.projectDraft = null;
    state.projectTemplateSelection = null;
    renderProjectTemplateOptions();
  }
}

export function openProjectTemplatePicker() {
  state.projectDraft = readProjectDraft();
  state.projectModalOpen = false;
  $("#projectModal").classList.add("hidden");
  state.templatePickerActive = true;
  switchView("templates");
  renderAll();
}

export function cancelProjectTemplatePicker() {
  state.templatePickerActive = false;
  state.templateModalOpen = false;
  switchView("projects");
  setProjectModal(true);
  renderAll();
}

export function clearProjectTemplateSelection() {
  state.projectTemplateSelection = null;
  state.activeTemplateId = "";
  state.templateValues = {};
  renderProjectTemplateOptions();
}

export function chooseProjectTemplate(templateId) {
  const template = state.templates.find((item) => item.id === templateId);
  if (!template?.body) {
    showToast("This template cannot be applied to a project");
    return;
  }
  state.templateApplyFormat = template.format || "yaml";
  setTemplateModal(true, templateId);
}

function readProjectDraft() {
  return {
    name: $("#projectName").value,
    repoUrl: $("#projectRepo").value,
    environments: $("#projectEnvironments").value,
    description: $("#projectDescription").value,
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

export function setSharedConfigCreateModal(open) {
  state.sharedConfigCreateModalOpen = open;
  renderSharedConfigCreateModal();
  if (open) {
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
  if (!window.confirm(`Delete global shared config ${item.name}?`)) return;
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

export function setHistoryModal(open) {
  state.historyModalOpen = open;
  if (!open) {
    state.historyLoading = false;
  }
  renderVersionHistory();
}

export function setReviewModal(open) {
  state.reviewModalOpen = open;
  renderReviewModal();
  if (open) {
    window.setTimeout(() => $("#reviewReason").focus(), 0);
  }
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
  if (state.templatePickerActive) {
    selectTemplateForProject(template);
    return;
  }

  const project = activeProject();
  if (!template || !project) {
    showToast("Choose a template and project first");
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
      configFileId: state.activeConfigFile,
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
    configFileId: state.activeConfigFile,
  };
  state.templateModalOpen = false;
  state.importPreviewOpen = true;
  renderAll();
  showToast(`Extracted ${result.entryCount} config keys`);
}

function selectTemplateForProject(template) {
  if (!template) {
    showToast("Choose a template first");
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

  const outputFormat =
    $("#templateApplyFormat")?.value ||
    state.templateApplyFormat ||
    template.format ||
    "yaml";
  state.templateApplyFormat = outputFormat;
  state.projectTemplateSelection = {
    templateId: template.id,
    templateName: template.name,
    sourceFormat: template.format || "yaml",
    outputFormat,
    content: renderTemplateContent(template),
    values: { ...state.templateValues },
  };
  state.templatePickerActive = false;
  state.templateModalOpen = false;
  switchView("projects");
  setProjectModal(true);
  renderAll();
  showToast(`${template.name} selected`);
}

function renderTemplateContent(template) {
  return (template.body || "").replace(
    /\$\{([A-Z0-9_]+)\}/g,
    (_, name) => state.templateValues[name] ?? "",
  );
}

export function setExportModal(open) {
  const project = activeProject();
  state.exportModalOpen = open;
  if (open) {
    state.exportFormat = "yaml";
  }
  renderExportModal();
  if (open) {
    window.setTimeout(() => $("#exportFormat").focus(), 0);
  }
}

export function setImportModal(open) {
  state.importModalOpen = open;
  $("#importModal").classList.toggle("hidden", !open);
  if (open) {
    window.setTimeout(() => $("#configFile").focus(), 0);
  } else {
    $("#configFile").value = "";
  }
}

export function setImportPreviewModal(open) {
  state.importPreviewOpen = open;
  if (!open && !state.importApplying) {
    state.importPreview = null;
  }
  renderImportPreview();
}

export async function openVersionHistory() {
  state.historyLoading = true;
  state.historyModalOpen = true;
  renderVersionHistory();
  try {
    await loadConfigHistory();
  } finally {
    state.historyLoading = false;
    renderVersionHistory();
  }
}

export async function createProject(event) {
  event.preventDefault();
  const templateSelection = state.projectTemplateSelection;
  const templateId = templateSelection?.templateId || "";
  const created = await api("/projects", {
    method: "POST",
    body: JSON.stringify({
      name: $("#projectName").value,
      repoUrl: $("#projectRepo").value,
      templateId,
      groupId: $("#projectGroup")?.value || "",
      environments: parseEnvironmentInput($("#projectEnvironments").value),
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
  }
  await loadConfigsAndHistory();
  setProjectModal(false);
  if (templateSelection) {
    const result = await api(`/projects/${created.id}/configs/extract`, {
      method: "POST",
      body: JSON.stringify({
        environment: state.activeEnvironment,
        format: templateSelection.sourceFormat,
        content: templateSelection.content,
      }),
    });
    const outputContent =
      templateSelection.outputFormat === templateSelection.sourceFormat
        ? templateSelection.content
        : serializeConfigs(
            result.entries || [],
            templateSelection.outputFormat,
          );
    state.importPreview = {
      ...result,
      fileName: templateSelection.templateName,
      content: outputContent,
      format: templateSelection.outputFormat,
      projectId: created.id,
      environment: state.activeEnvironment,
    };
    state.importPreviewOpen = true;
    switchView("config");
    renderAll();
    showToast(`${created.name} created. Review extracted template config.`);
    return;
  }
  switchView("projects");
  renderAll();
  showToast(`${created.name} created`);
}

export async function rollbackLatestVersion() {
  const project = activeProject();
  const previous = state.configHistory[1];
  if (!project || !previous) {
    showToast("No previous config revision to restore");
    return;
  }

  const confirmed = window.confirm(
    `Rollback ${project.name} ${state.activeEnvironment} config to the previous revision?`,
  );
  if (!confirmed) return;

  await api(`/projects/${project.id}/config-history/rollback`, {
    method: "POST",
    body: JSON.stringify({
      environment: state.activeEnvironment,
      revisionId: previous.id,
      changeReason: "rollback config revision from frontend history",
    }),
  });

  await loadConfigsAndHistory();
  renderAll();
  showToast(`${project.name} ${state.activeEnvironment} config rolled back`);
}

export function startInlineEdit(configId, field) {
  const config = state.configs.find((entry) => entry.id === configId);
  if (!config) return;

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

  state.inlineSaving = true;
  try {
    const body =
      edit.field === "key"
        ? { key: nextValue, changeReason: "inline key edit from frontend" }
        : { value: nextValue, changeReason: "inline value edit from frontend" };
    const updated = await api(
      `/projects/${config.projectId}/configs/${config.id}`,
      {
        method: "PUT",
        body: JSON.stringify(body),
      },
    );
    state.inlineEdit = null;
    await loadConfigsAndHistory();
    syncPendingReviewChanges();
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
    }));
}

function configHasNetChange(config) {
  const baseline = state.configBaseline.get(config.id);
  if (!baseline) return true;
  return baseline.key !== config.key || baseline.value !== config.value;
}

export async function hasReviewRequest(config) {
  const requests = await api(
    `/projects/${config.projectId}/review-requests?env=prod&key=${encodeURIComponent(
      config.key,
    )}&status=pending`,
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

  const format = $("#configFormat").value;
  const content = await file.text();
  const result = await api(`/projects/${project.id}/configs/extract`, {
    method: "POST",
    body: JSON.stringify({
      environment: state.activeEnvironment,
      configFileId: state.activeConfigFile,
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
    configFileId: state.activeConfigFile,
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
        configFileId: preview.configFileId || state.activeConfigFile,
        format: preview.format,
        content: preview.content,
        changeReason: `import ${preview.fileName}`,
      }),
    });

    await loadConfigsAndHistory();
    syncPendingReviewChanges();
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
      `/projects/${project.id}/configs?env=${encodeURIComponent(state.activeEnvironment)}&revealSensitive=true`,
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
    `${project.name}-${state.activeEnvironment}.${extension}`,
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

  await api("/review-requests", {
    method: "POST",
    body: JSON.stringify({
      projectId: project.id,
      environment: state.activeEnvironment,
      configKey: pending.key,
      reason,
    }),
  });

  state.requests = await api("/review-requests");
  acceptCurrentConfigsAsBaseline();
  state.reviewModalOpen = false;
  renderAll();
  showToast("Review request created");
}

export async function handleReviewDecision(id, action) {
  await api(`/review-requests/${id}/${action}`, {
    method: "PUT",
    body: JSON.stringify({ comment: `${action} from frontend` }),
  });
  state.requests = await api("/review-requests");
  renderDashboard();
  renderRequests();
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
