import {
  cancelInlineEdit,
  commitInlineEdit,
  createConfigFile,
  createConfigKey,
  createGroup,
  createProject,
  createTemplate,
  addProjectMembers,
  createReviewRequest,
  createSharedConfig,
  updateSharedConfig,
  exportCurrentConfig,
  handleReviewDecision,
  extractConfigFile,
  addGroupMember,
  applyImportPreview,
  applyTemplate,
  openExportConfig,
  openGroupPanel,
  openConfigSourcePicker,
  openVersionHistory,
  importConfigSource,
  rollbackConfigRevision,
  setConfigCreateDrawer,
  setActiveBranch,
  setCompareEnvironment,
  setConfigFileCreate,
  setConfigFileMenu,
  setConfigMode,
  setExportModal,
  removeGroupMember,
  removeProjectMember,
  removeSelectedCreateMember,
  removeSelectedGroupMember,
  removeSelectedProjectMember,
  selectGroup,
  setGroupDetailTab,
  setGroupModal,
  setHistoryModal,
  setNotificationPopover,
  setLibraryTab,
  setImportModal,
  setImportPreviewModal,
  setProjectDetailTab,
  setProjectModal,
  setReviewModal,
  setRequestDetailModal,
  setTemplateCreateModal,
  setSharedConfigCreateModal,
  setSharedConfigEditModal,
  setTemplateModal,
  startInlineEdit,
  toggleGroupCreate,
  toggleGroupCreateMemberSelection,
  toggleGroupMemberPicker,
  toggleConfigFileMenu,
  toggleProjectMemberPicker,
  toggleProjectMemberSelection,
  toggleProjectRoleMenu,
  toggleGroupMemberSelection,
  submitReviewChanges,
  switchView,
  updateGroupCreateMemberSearch,
  updateConfigFileDraftName,
  updateConfigFileSourceType,
  updateGroupMemberRole,
  updateProjectMemberRole,
  updateProjectMemberSearch,
  updateGroupMemberSearch,
  updateNewConfigValueType,
  updateTemplateValue,
  toggleSensitiveReveal,
  deleteSharedConfig,
  submitSharedConfigUpdate,
  openSharedConfigEdit,
} from "./actions.js";
import { api } from "./api.js";
import { loadCompareConfigs, loadConfigsAndHistory, loadInitialData } from "./data.js";
import { $, setAuthenticated, showToast } from "./dom.js";
import { hydrateHistoryRevisionDetails, renderAll, renderConfigRows } from "./render.js";
import { activeProject, state } from "./state.js";

export function bindEvents() {
  document.addEventListener("click", handleDocumentClick);
  document.addEventListener("toggle", handleDocumentToggle, true);
  document.addEventListener("keydown", handleDocumentKeydown);
  document.addEventListener("focusout", handleDocumentFocusOut);

  $("#loginForm").addEventListener("submit", handleLogin);
  $("#globalSearch").addEventListener("input", (event) => {
    state.globalSearch = event.target.value;
    renderAll();
  });

  $("#configSearch").addEventListener("input", (event) => {
    state.configSearch = event.target.value;
    renderConfigRows();
  });

  document.addEventListener("input", (event) => {
    const configFileName = event.target.closest("#newConfigFileName");
    if (configFileName) {
      updateConfigFileDraftName(configFileName.value);
      return;
    }

    const groupCreateSearch = event.target.closest("#groupCreateMemberSearch");
    if (groupCreateSearch) {
      updateGroupCreateMemberSearch(groupCreateSearch.value);
      return;
    }

    const groupMemberSearch = event.target.closest("#groupMemberSearch");
    if (groupMemberSearch) {
      updateGroupMemberSearch(groupMemberSearch.value);
      return;
    }

    const projectMemberSearch = event.target.closest("#projectMemberSearch");
    if (projectMemberSearch) {
      updateProjectMemberSearch(projectMemberSearch.value);
      return;
    }

    const reviewReason = event.target.closest("#reviewReason");
    if (reviewReason) {
      reviewReason.dataset.touched = "true";
      return;
    }

    const target = event.target.closest("[data-template-variable]");
    if (!target) return;
    updateTemplateValue(target.dataset.templateVariable, target.value);
  });

  document.addEventListener("change", (event) => {
    const createMember = event.target.closest(
      "[data-group-create-member-option]",
    );
    if (createMember) {
      toggleGroupCreateMemberSelection(
        createMember.value,
        createMember.checked,
      );
      return;
    }

    const member = event.target.closest("[data-group-member-option]");
    if (member) {
      toggleGroupMemberSelection(member.value, member.checked);
      return;
    }

    const projectMember = event.target.closest("[data-project-member-option]");
    if (projectMember) {
      toggleProjectMemberSelection(projectMember.value, projectMember.checked);
      return;
    }

    const branchSelect = event.target.closest("[data-branch-select]");
    if (branchSelect) {
      setActiveBranch(branchSelect.value).catch((error) => showToast(error.message));
      return;
    }

    const compareEnv = event.target.closest("[data-compare-env]");
    if (compareEnv) {
      setCompareEnvironment(
        compareEnv.dataset.compareEnv,
        compareEnv.value,
      ).catch((error) => showToast(error.message));
      return;
    }

    const sharedConfigScope = event.target.closest("#sharedConfigScope");
    if (sharedConfigScope) {
      const groupField = document.querySelector("#sharedConfigGroupField");
      groupField?.classList.toggle("hidden", sharedConfigScope.value !== "group");
      return;
    }

    const newConfigValueType = event.target.closest("#newConfigValueType");
    if (newConfigValueType) {
      updateNewConfigValueType(newConfigValueType.value);
      return;
    }

    const targetTemplateFormat = event.target.closest("#templateApplyFormat");
    if (!targetTemplateFormat) return;
    state.templateApplyFormat = targetTemplateFormat.value;
  });

  $("#configFile").addEventListener("change", (event) => {
    const name = event.target.files[0]?.name || "";
    const ext = name.split(".").pop()?.toLowerCase();
    if (ext === "json") $("#configFormat").value = "json";
    if (ext === "yaml" || ext === "yml") $("#configFormat").value = "yaml";
    if (ext === "properties") $("#configFormat").value = "properties";
  });

  $("#importConfig").addEventListener("click", () => {
    extractConfigFile().catch((error) => showToast(error.message));
  });

  $("#submitReview").addEventListener("click", () => {
    createReviewRequest().catch((error) => showToast(error.message));
  });

  $("#exportConfig").addEventListener("click", () => {
    openExportConfig();
  });

  $("#projectForm").addEventListener("submit", (event) => {
    createProject(event).catch((error) => showToast(error.message));
  });

  $("#templateCreateForm").addEventListener("submit", (event) => {
    createTemplate(event).catch((error) => showToast(error.message));
  });

  $("#sharedConfigCreateForm").addEventListener("submit", (event) => {
    createSharedConfig(event).catch((error) => showToast(error.message));
  });

  $("#sharedConfigEditForm").addEventListener("submit", (event) => {
    updateSharedConfig(event).catch((error) => showToast(error.message));
  });

  $("#configCreateForm").addEventListener("submit", (event) => {
    createConfigKey(event).catch((error) => showToast(error.message));
  });

  document.addEventListener("submit", (event) => {
    const configFileForm = event.target.closest("#configFileCreateForm");
    if (configFileForm) {
      createConfigFile(event).catch((error) => showToast(error.message));
    }
  });

  $("#groupForm").addEventListener("submit", (event) => {
    createGroup(event).catch((error) => showToast(error.message));
  });
}

export function initApp() {
  bindEvents();

  if (state.token && state.user) {
    setAuthenticated(true);
    loadInitialData()
      .then(renderAll)
      .catch((error) => {
        showToast(error.message);
        setAuthenticated(false);
      });
    return;
  }

  setAuthenticated(false);
}

async function handleDocumentClick(event) {
  const target = event.target.closest("button");

  if (!event.target.closest(".group-role-dropdown-container")) {
    const { state } = await import("./state.js");
    if (state.groupRoleMenuUserId) {
      const { toggleGroupRoleMenu } = await import("./actions.js");
      toggleGroupRoleMenu("");
    }
    if (state.projectRoleMenuUserId) {
      toggleProjectRoleMenu("");
    }
  }

  if (!event.target.closest(".config-file-add-wrap") && state.configFileMenuOpen) {
    setConfigFileMenu(false);
  }

  if (!event.target.closest(".notification-wrap") && state.notificationPopoverOpen) {
    setNotificationPopover(false);
  }

  const requestItem = event.target.closest("[data-open-request]");
  if (requestItem && !event.target.closest("button")) {
    setRequestDetailModal(true, requestItem.dataset.openRequest);
    return;
  }

  if (!target) return;

  try {
    if (target.id === "notificationButton") {
      setNotificationPopover(!state.notificationPopoverOpen);
      return;
    }

    if (target.dataset.userMenuAction === "groups") {
      await openGroupPanel();
      return;
    }

    if (target.dataset.userMenuAction === "logout") {
      handleLogout();
      return;
    }

    if (target.dataset.closeModal === "group") {
      setGroupModal(false);
      return;
    }

    if (target.dataset.closeDrawer === "config-create") {
      setConfigCreateDrawer(false);
      return;
    }

    if (target.dataset.selectGroup) {
      await selectGroup(target.dataset.selectGroup);
      return;
    }

    if (target.dataset.groupTab) {
      setGroupDetailTab(target.dataset.groupTab);
      return;
    }

    if (target.dataset.libraryTab) {
      setLibraryTab(target.dataset.libraryTab);
      return;
    }

    if (target.id === "openGroupCreate") {
      toggleGroupCreate(true);
      return;
    }

    if (target.id === "cancelGroupCreate") {
      toggleGroupCreate(false);
      return;
    }

    if (
      target.id === "openGroupMemberPicker" ||
      target.closest("#closeGroupMemberPicker")
    ) {
      toggleGroupMemberPicker();
      return;
    }

    if (target.id === "addGroupMember") {
      await addGroupMember();
      return;
    }

    if (target.dataset.removeSelectedCreateMember) {
      removeSelectedCreateMember(target.dataset.removeSelectedCreateMember);
      return;
    }

    if (target.dataset.removeSelectedGroupMember) {
      removeSelectedGroupMember(target.dataset.removeSelectedGroupMember);
      return;
    }

    if (target.dataset.removeSelectedProjectMember) {
      removeSelectedProjectMember(target.dataset.removeSelectedProjectMember);
      return;
    }

    if (target.dataset.removeGroupMember) {
      await removeGroupMember(target.dataset.removeGroupMember);
      return;
    }

    if (target.dataset.toggleRoleMenu) {
      const { toggleGroupRoleMenu } = await import("./actions.js");
      toggleGroupRoleMenu(target.dataset.toggleRoleMenu);
      return;
    }

    if (target.dataset.setRole) {
      const { updateGroupMemberRole } = await import("./actions.js");
      await updateGroupMemberRole(
        target.dataset.userId,
        target.dataset.setRole,
      );
      return;
    }

    if (target.dataset.toggleProjectRoleMenu) {
      toggleProjectRoleMenu(target.dataset.toggleProjectRoleMenu);
      return;
    }

    if (target.dataset.setProjectRole) {
      await updateProjectMemberRole(
        target.dataset.userId,
        target.dataset.setProjectRole,
      );
      return;
    }

    if (target.dataset.closeModal === "project") {
      setProjectModal(false);
      return;
    }

    if (target.dataset.closeModal === "history") {
      setHistoryModal(false);
      return;
    }

    if (target.dataset.closeModal === "request-detail") {
      setRequestDetailModal(false);
      return;
    }

    if (target.dataset.closeModal === "export") {
      setExportModal(false);
      return;
    }

    if (target.dataset.closeModal === "import") {
      setImportModal(false);
      return;
    }

    if (target.dataset.closeModal === "import-preview") {
      setImportPreviewModal(false);
      return;
    }

    if (target.dataset.closeModal === "config-source") {
      openConfigSourcePicker("");
      return;
    }

    if (target.dataset.closeModal === "review") {
      setReviewModal(false);
      return;
    }

    if (target.dataset.closeModal === "template") {
      setTemplateModal(false);
      return;
    }

    if (target.dataset.closeModal === "template-create") {
      setTemplateCreateModal(false);
      return;
    }

    if (target.dataset.closeModal === "shared-config-create") {
      setSharedConfigCreateModal(false);
      return;
    }

    if (target.dataset.closeModal === "shared-config-edit") {
      setSharedConfigEditModal(false);
      return;
    }

    if (target.id === "registerProject") {
      setProjectModal(true);
      return;
    }

    if (target.id === "openConfigCreate") {
      setConfigCreateDrawer(true);
      return;
    }

    if (target.id === "openConfigFileCreate") {
      toggleConfigFileMenu();
      return;
    }

    if (target.dataset.configFileMenuAction === "blank") {
      setConfigFileCreate(true);
      return;
    }

    if (target.dataset.configFileMenuAction) {
      openConfigSourcePicker(target.dataset.configFileMenuAction);
      return;
    }

    if (target.dataset.configSourceType) {
      updateConfigFileSourceType(target.dataset.configSourceType);
      return;
    }

    if (target.dataset.openConfigSourcePicker) {
      openConfigSourcePicker(target.dataset.openConfigSourcePicker);
      return;
    }

    if (target.dataset.importConfigSource) {
      await importConfigSource(
        target.dataset.importConfigSource,
        target.dataset.sourceId,
      );
      return;
    }

    if (target.dataset.configMode) {
      await setConfigMode(target.dataset.configMode);
      return;
    }

    if (target.dataset.projectTab) {
      await setProjectDetailTab(target.dataset.projectTab);
      return;
    }

    if (target.id === "openProjectMemberPicker" || target.id === "closeProjectMemberPicker") {
      toggleProjectMemberPicker();
      return;
    }

    if (target.id === "addProjectMember") {
      await addProjectMembers();
      return;
    }

    if (target.dataset.removeProjectMember) {
      await removeProjectMember(target.dataset.removeProjectMember);
      return;
    }

    if (target.id === "backToProjects") {
      switchView("projects");
      renderAll();
      return;
    }

    if (target.id === "openTemplateCreate") {
      if (state.libraryTab === "shared-config") {
        setSharedConfigCreateModal(true);
        return;
      }
      setTemplateCreateModal(true);
      return;
    }

    if (target.dataset.editSharedConfig) {
      openSharedConfigEdit(target.dataset.editSharedConfig);
      return;
    }

    if (target.dataset.deleteSharedConfig) {
      await deleteSharedConfig(target.dataset.deleteSharedConfig);
      return;
    }

    if (target.dataset.submitSharedConfig) {
      await submitSharedConfigUpdate(target.dataset.submitSharedConfig);
      return;
    }

    if (target.id === "openImportConfig") {
      setImportModal(true);
      return;
    }

    if (target.dataset.rollbackRevision) {
      event.preventDefault();
      event.stopPropagation();
      await rollbackConfigRevision(target.dataset.rollbackRevision);
      return;
    }

    if (target.id === "applyImportConfig") {
      await applyImportPreview();
      return;
    }

    if (target.id === "confirmSubmitReview") {
      await submitReviewChanges();
      return;
    }

    if (target.id === "confirmExportConfig") {
      await exportCurrentConfig();
      return;
    }

    if (target.id === "confirmApplyTemplate") {
      await applyTemplate();
      return;
    }

    const jump = target.dataset.jump;
    if (jump) {
      setNotificationPopover(false);
      switchView(jump);
      if (target.dataset.openProjectForm) {
        setProjectModal(true);
      }
      return;
    }

    const viewTarget = target.dataset.viewTarget;
    if (viewTarget) {
      setNotificationPopover(false);
      switchView(viewTarget);
      return;
    }

    if (target.dataset.selectConfigFile) {
      state.activeConfigFile = target.dataset.selectConfigFile;
      await loadConfigsAndHistory();
      renderAll();
      return;
    }

    const projectId = target.dataset.openConfig ?? target.dataset.selectProject;
    if (projectId) {
      state.activeProjectId = projectId;
      state.activeConfigFile = "";
      state.projectDetailTab = "configs";
      state.projectMemberPickerOpen = false;
      const project = activeProject();
      state.activeEnvironment = project.environments.includes("prod")
        ? "prod"
        : project.environments[0];
      state.activeBranch = project.branches?.includes("default")
        ? "default"
        : project.branches?.[0] || "default";
      await loadConfigsAndHistory();
      state.compareConfigs = {};
      if (state.configMode === "compare") {
        await loadCompareConfigs();
      }
      switchView("config");
      renderAll();
      return;
    }

    if (target.dataset.env) {
      state.activeEnvironment = target.dataset.env;
      await loadConfigsAndHistory();
      renderAll();
      return;
    }

    if (target.dataset.reveal) {
      await toggleSensitiveReveal(target.dataset.reveal);
      return;
    }

    if (
      target.id === "openConfigHistory" ||
      target.id === "dashboardHistoryAction"
    ) {
      await openVersionHistory();
      return;
    }

    if (target.dataset.startInlineEdit) {
      startInlineEdit(target.dataset.startInlineEdit, target.dataset.field);
      return;
    }

    if (target.dataset.approve) {
      await handleReviewDecision(target.dataset.approve, "approve");
      return;
    }

    if (target.dataset.reject) {
      await handleReviewDecision(target.dataset.reject, "reject");
      return;
    }

    if (target.dataset.viewRequest) {
      setRequestDetailModal(true, target.dataset.viewRequest);
      return;
    }
  } catch (error) {
    showToast(error.message);
  }
}

function handleDocumentToggle(event) {
  const details = event.target.closest?.(".version-timeline-item");
  if (!details || !details.open) return;
  hydrateHistoryRevisionDetails(details);
}

async function handleDocumentKeydown(event) {
  if (event.key === "Escape" && event.target.closest("#configFileCreateForm")) {
    event.preventDefault();
    setConfigFileCreate(false);
    return;
  }

  if (event.key === "Escape" && state.configFileMenuOpen) {
    event.preventDefault();
    setConfigFileMenu(false);
    return;
  }

  if (event.key === "Escape" && state.configSourceModalOpen) {
    event.preventDefault();
    openConfigSourcePicker("");
    return;
  }

  const input = event.target.closest("[data-inline-input]");
  if (!input) return;

  if (event.key === "Enter") {
    event.preventDefault();
    try {
      await commitInlineEdit(input);
    } catch (error) {
      showToast(error.message);
    }
    return;
  }

  if (event.key === "Escape") {
    event.preventDefault();
    cancelInlineEdit();
  }
}

function handleDocumentFocusOut(event) {
  const input = event.target.closest("[data-inline-input]");
  if (!input) return;

  window.setTimeout(() => {
    if (document.activeElement?.dataset?.inlineInput) return;
    commitInlineEdit(input).catch((error) => showToast(error.message));
  }, 0);
}

async function handleLogin(event) {
  event.preventDefault();
  try {
    const data = await api("/auth/login", {
      method: "POST",
      body: JSON.stringify({
        email: $("#loginEmail").value,
        password: $("#loginPassword").value,
      }),
    });
    state.token = data.token;
    state.user = data.user;
    localStorage.setItem("configManToken", state.token);
    localStorage.setItem("configManUser", JSON.stringify(state.user));
    setAuthenticated(true);
    await loadInitialData();
    renderAll();
    showToast(`Signed in as ${state.user.name}`);
  } catch (error) {
    showToast(error.message);
  }
}

function handleLogout() {
  localStorage.removeItem("configManToken");
  localStorage.removeItem("configManUser");
  state.token = "";
  state.user = null;
  setAuthenticated(false);
}
