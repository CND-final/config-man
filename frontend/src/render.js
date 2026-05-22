import { $ } from "./dom.js";
import {
  configFileForEntry,
  configFilesForEntries,
  configsForActiveFile,
  ensureActiveConfigFile,
} from "./configFiles.js";
import { activeProject, navItems, state } from "./state.js";
import { escapeHtml, formatDateTime, initials, statusClass } from "./utils.js";

function searchTerm() {
  return state.globalSearch.trim().toLowerCase();
}

function includesSearch(values, term = searchTerm()) {
  if (!term) return true;
  return values.some((value) =>
    String(value ?? "")
      .toLowerCase()
      .includes(term),
  );
}

function filteredProjects() {
  const term = searchTerm();
  return state.projects.filter((project) =>
    includesSearch(
      [
        project.name,
        project.owner,
        project.repoUrl,
        project.defaultFormat,
        projectTemplateName(project.templateId),
        ...project.environments,
      ],
      term,
    ),
  );
}

function filteredTemplates() {
  const term = searchTerm();
  return state.templates.filter((template) =>
    includesSearch(
      [
        template.name,
        template.description,
        template.format,
        template.isCustom ? "personal" : "shared",
        ...template.keys,
      ],
      term,
    ),
  );
}

function filteredRequests() {
  const term = searchTerm();
  return state.requests.filter((request) =>
    includesSearch(
      [
        request.id,
        request.projectName,
        request.requester,
        request.environment,
        request.configKey,
        request.reason,
        request.status,
      ],
      term,
    ),
  );
}

function matchesConfigSearch(config) {
  const local = state.configSearch.trim().toLowerCase();
  const global = searchTerm();
  const file = configFileForEntry(config);
  const values = [
    config.key,
    config.value,
    config.environment,
    config.updatedBy,
    config.valueType,
    file.name,
  ];
  return includesSearch(values, local) && includesSearch(values, global);
}

export function renderNav() {
  $("#navList").innerHTML = navItems
    .map(
      (item) => `
        <button class="nav-item ${state.activeView === item.id ? "active" : ""}" type="button" data-view-target="${item.id}">
          <img class="nav-icon" src="${item.icon}" alt="" aria-hidden="true" />
          <span class="nav-label">${item.label}</span>
        </button>
      `,
    )
    .join("");
}

export function renderUser() {
  $("#userInitials").textContent = initials(state.user?.name || "--");
  $("#userName").textContent =
    `${state.user?.name || "Not signed in"} · ${state.user?.role || ""}`;
}

export function renderStats() {
  const pendingCount = state.requests.filter(
    (request) => request.status === "pending",
  ).length;
  const sensitiveCount = state.configs.filter(
    (config) => config.isSensitive,
  ).length;
  const stats = [
    {
      label: "Projects",
      value: state.projects.length,
      change: "Live from API",
    },
    {
      label: "Active Keys",
      value: state.configs.length,
      change: state.activeEnvironment,
    },
    { label: "Pending Reviews", value: pendingCount, change: "Prod guarded" },
    {
      label: "Sensitive Keys",
      value: sensitiveCount,
      change: "Masked by default",
    },
  ];

  $("#statsGrid").innerHTML = stats
    .map(
      (stat) => `
        <article class="stat-card">
          <p>${stat.label}</p>
          <strong>${stat.value}</strong>
          <span class="metric-change">${stat.change}</span>
        </article>
      `,
    )
    .join("");
}

export function renderDashboard() {
  renderStats();
  const dashboardProjects = filteredProjects().slice(0, 3);
  $("#dashboardProjects").innerHTML =
    dashboardProjects
      .map(
        (project) => `
        <article class="project-row">
          <div>
            <h3>${escapeHtml(project.name)}</h3>
            <p class="project-meta">
              <span>${escapeHtml(project.owner)}</span>
              <span>${project.configCount} keys</span>
              <span>${escapeHtml(project.lastChanged)}</span>
            </p>
          </div>
        </article>
      `,
      )
      .join("") || '<p class="project-meta">No matching projects.</p>';

  $("#dashboardRequests").innerHTML =
    filteredRequests()
      .filter((request) => request.status === "pending")
      .slice(0, 4)
      .map(
        (request) => `
          <article class="project-row">
            <div>
              <h3>${escapeHtml(request.id.slice(0, 8))}</h3>
              <p class="project-meta">
                <span>${escapeHtml(request.projectName)}</span>
                <span>${escapeHtml(request.requester)}</span>
              </p>
            </div>
            <span class="status-pill warning">pending</span>
          </article>
        `,
      )
      .join("") || '<p class="project-meta">No pending review requests.</p>';
}

export function renderProjects() {
  $("#projectsGrid").innerHTML =
    filteredProjects()
      .map(
        (project) => `
          <button class="project-card project-card-button" type="button" data-open-config="${project.id}">
            <div class="card-top">
              <div class="card-title">
                <h3>${escapeHtml(project.name)}</h3>
                <p>${escapeHtml(project.repoUrl || "No repository URL")}</p>
              </div>
            </div>
            <p class="project-meta">
              <span>${escapeHtml(project.owner)}</span>
              <span>${project.configCount} config keys</span>
              <span>${escapeHtml(project.defaultFormat)}</span>
              ${projectTemplateName(project.templateId) ? `<span>${escapeHtml(projectTemplateName(project.templateId))}</span>` : ""}
            </p>
            <div class="environment-strip">
              ${project.environments.map((environment) => `<span>${environment}</span>`).join("")}
            </div>
          </button>
        `,
      )
      .join("") || "<p>No matching projects.</p>";
}

export function renderTemplates() {
  const action = $("#openTemplateCreate");
  if (action) {
    action.textContent = state.templatePickerActive ? "Cancel" : "New Template";
    action.className = state.templatePickerActive
      ? "secondary-action"
      : "primary-action";
  }

  $("#templatesGrid").innerHTML =
    filteredTemplates().map(renderTemplateCard).join("") ||
    '<p class="project-meta">No matching templates.</p>';
  renderTemplateModal();
  renderTemplateCreateModal();
}

function renderTemplateCard(template) {
  const canPick = state.templatePickerActive && template.body;
  const tag = canPick ? "button" : "article";
  const typeAttr = canPick ? ' type="button"' : "";
  const pickAttr = canPick
    ? ` data-pick-template="${escapeHtml(template.id)}"`
    : "";
  return `
    <${tag} class="template-card ${canPick ? "template-card-button" : ""}"${typeAttr}${pickAttr}>
      <div class="card-top">
        <div class="card-title">
          <h3>${escapeHtml(template.name)}</h3>
          <p>${escapeHtml(template.description || "Reusable configuration template")}</p>
        </div>
        <div class="template-badges">
          <span class="status-pill neutral">${template.isCustom ? "personal" : "shared"}</span>
        </div>
      </div>
      <pre class="template-body-preview">${escapeHtml(template.body || template.entries?.map((entry) => `${entry.key}=${entry.defaultValue}`).join("\n") || "")}</pre>
      <div class="template-list">
        ${renderTemplateKeys(template)}
      </div>
    </${tag}>
  `;
}

function renderTemplateKeys(template) {
  const values = template.variables?.length
    ? template.variables.map((variable) => variable.name)
    : template.keys;
  return values
    .map(
      (key) => `
        <div class="template-key">
          <span>${escapeHtml(key)}</span>
        </div>
      `,
    )
    .join("");
}

function projectTemplateName(templateId) {
  if (!templateId) return "";
  return (
    state.templates.find((template) => template.id === templateId)?.name || ""
  );
}

export function renderConfigFileList() {
  ensureActiveConfigFile(state);
  $("#configProjectList").innerHTML = configFilesForEntries(state.configs)
    .map(
      (file) => `
        <button class="compact-item config-file-item ${file.id === state.activeConfigFile ? "active" : ""}" type="button" data-select-config-file="${escapeHtml(file.id)}">
          <div>
            <h3>${escapeHtml(file.name)}</h3>
            <p class="project-meta">
              <span>${file.count} ${file.count === 1 ? "key" : "keys"}</span>
              <span>${escapeHtml(file.detail)}</span>
            </p>
          </div>
        </button>
      `,
    )
    .join("");
}

export function renderEnvironmentTabs() {
  const project = activeProject();
  if (!project) {
    $("#environmentTabs").innerHTML = "";
    return;
  }
  if (!project.environments.includes(state.activeEnvironment)) {
    state.activeEnvironment = project.environments[0];
  }

  $("#environmentTabs").innerHTML = project.environments
    .map(
      (environment) => `
        <button class="${environment === state.activeEnvironment ? "active" : ""}" type="button" data-env="${environment}">
          ${environment}
        </button>
      `,
    )
    .join("");
}

export function renderProjectTemplateOptions() {
  const summary = $("#projectTemplateSelection");
  const clearButton = $("#clearProjectTemplate");
  if (!summary) return;

  const selection = state.projectTemplateSelection;
  summary.textContent = selection
    ? `${selection.templateName} · ${selection.outputFormat}`
    : "No template selected.";
  if (clearButton) {
    clearButton.classList.toggle("hidden", !selection);
  }
}

export function renderConfigRows(renderShell = true) {
  const project = activeProject();
  if (renderShell) {
    $("#configTitle").textContent = project?.name || "Project Config";
    renderConfigVersionLabel();
    renderConfigFileList();
    renderEnvironmentTabs();
  }

  const rows = configsForActiveFile(
    state.configs,
    state.activeConfigFile,
  ).filter(matchesConfigSearch);

  $("#configRows").innerHTML =
    rows.map(renderConfigRowMarkup).join("") ||
    `<tr><td colspan="3" class="value-cell">No config keys match this view.</td></tr>`;
  renderReviewDock();
}

export function renderConfigRow(configId) {
  const config = state.configs.find((entry) => entry.id === configId);
  const row = Array.from(document.querySelectorAll("[data-config-row]")).find(
    (element) => element.dataset.configRow === configId,
  );

  if (
    !config ||
    !row ||
    !configsForActiveFile([config], state.activeConfigFile).length ||
    !matchesConfigSearch(config)
  ) {
    renderConfigRows(false);
    return;
  }

  row.outerHTML = renderConfigRowMarkup(config);
  renderReviewDock();
}

function renderConfigRowMarkup(config) {
  const revealKey = `${config.projectId}:${config.environment}:${config.key}`;
  const valueIsRevealed = config.isSensitive && state.revealedKeys.has(revealKey);
  const valueIsMasked = config.isSensitive && !valueIsRevealed;
  const visibleValue = valueIsMasked ? "******" : config.value;
  return `
    <tr data-config-row="${escapeHtml(config.id)}">
      ${renderEditableConfigCell(config, "key", config.key, "key-cell")}
      ${renderEditableConfigCell(config, "value", visibleValue, "value-cell", valueIsMasked, revealKey, valueIsRevealed)}
      <td>${escapeHtml(config.updated)}</td>
    </tr>
  `;
}

function renderEditableConfigCell(
  config,
  field,
  value,
  className,
  valueIsMasked = false,
  revealKey = "",
  valueIsRevealed = false,
) {
  const isEditing =
    state.inlineEdit?.configId === config.id &&
    state.inlineEdit?.field === field;
  const label = field === "key" ? "Edit key" : "Edit value";
  const inputType =
    field === "value" && config.isSensitive ? "password" : "text";

  if (isEditing) {
    return `
      <td class="${className} editable-cell editing-cell">
        <span class="editable-value editing-placeholder">${escapeHtml(state.inlineEdit.value)}</span>
        <input
          class="inline-edit-input"
          type="${inputType}"
          value="${escapeHtml(state.inlineEdit.value)}"
          data-inline-input="true"
          data-config-id="${escapeHtml(config.id)}"
          data-field="${field}"
          aria-label="${label}"
        />
      </td>
    `;
  }

  return `
    <td class="${className} editable-cell">
      <span class="editable-value">${escapeHtml(value)}</span>
      <span class="cell-tools">
        ${renderRevealToggle(config, valueIsMasked, valueIsRevealed, revealKey)}
        <button class="cell-icon-button cell-edit-button" type="button" data-start-inline-edit="${escapeHtml(config.id)}" data-field="${field}" aria-label="${label}" title="${label}">
          <span class="pencil-icon" aria-hidden="true"></span>
        </button>
      </span>
    </td>
  `;
}

function renderRevealToggle(config, valueIsMasked, valueIsRevealed, revealKey) {
  if (!config.isSensitive || !revealKey) return "";

  const label = valueIsMasked ? "Reveal sensitive value" : "Hide sensitive value";
  const iconClass = valueIsMasked ? "eye-icon eye-icon-off" : "eye-icon";
  return `
    <button
      class="cell-icon-button reveal-toggle"
      type="button"
      data-reveal="${escapeHtml(revealKey)}"
      aria-label="${label}"
      aria-pressed="${valueIsRevealed ? "true" : "false"}"
      title="${label}"
    >
      <span class="${iconClass}" aria-hidden="true"><span></span></span>
    </button>
  `;
}

export function renderReviewDock() {
  const dock = $("#reviewDock");
  if (!dock) return;
  const count = state.pendingReviewChanges.length;
  dock.classList.toggle("hidden", count === 0);
  $("#reviewChangeCount").textContent =
    `${count} ${count === 1 ? "change" : "changes"}`;
}

export function renderRequests() {
  $("#notificationCount").textContent = String(
    state.requests.filter((request) => request.status === "pending").length,
  );
  $("#requestList").innerHTML =
    filteredRequests()
      .map(
        (request) => `
          <article class="request-item">
            <div>
              <div class="project-meta">
                <span>${escapeHtml(request.id.slice(0, 8))}</span>
                <span>${escapeHtml(request.projectName)}</span>
                <span>${escapeHtml(request.environment)}</span>
                ${request.configKey ? `<span>${escapeHtml(request.configKey)}</span>` : ""}
              </div>
              <h3>${escapeHtml(request.reason)}</h3>
              <p>${escapeHtml(request.requester)}</p>
            </div>
            <div class="request-actions">
              <span class="status-pill ${statusClass(request.status)}">${escapeHtml(request.status)}</span>
              ${
                request.status === "pending" &&
                ["system_admin", "reviewer"].includes(state.user?.role)
                  ? `<button class="secondary-action" type="button" data-approve="${escapeHtml(request.id)}">Approve</button>
                     <button class="ghost-action" type="button" data-reject="${escapeHtml(request.id)}">Reject</button>`
                  : ""
              }
            </div>
          </article>
        `,
      )
      .join("") || '<p class="project-meta">No review requests yet.</p>';
}

export function renderVersionHistory() {
  $("#historyModal").classList.toggle("hidden", !state.historyModalOpen);
  if (!state.historyModalOpen) return;

  const project = activeProject();
  const current = state.configHistory[0];
  const previous = state.configHistory[1];
  $("#historyTitle").textContent = `${project?.name || "Config"} History`;
  $("#historyMeta").innerHTML = project
    ? `<span>${escapeHtml(project.name)}</span><span>${escapeHtml(state.activeEnvironment)}</span>`
    : "";

  if (state.historyLoading) {
    $("#historySummary").innerHTML =
      '<p class="project-meta">Loading config history...</p>';
    $("#versionList").innerHTML = "";
    $("#rollbackLatest").disabled = true;
    return;
  }

  $("#rollbackLatest").disabled = !previous;
  $("#historySummary").innerHTML = current
    ? `
      <div class="history-previous current">
        <span>Current Config</span>
        <code>${current.entries.length} keys · ${escapeHtml(current.changeReason)}</code>
      </div>
      <div class="history-previous">
        <span>Previous Config</span>
        <code>${previous ? `${previous.entries.length} keys · ${escapeHtml(previous.changeReason)}` : "No previous revision"}</code>
      </div>
    `
    : '<p class="project-meta">No config history yet.</p>';

  $("#versionList").innerHTML = state.configHistory
    .map((revision, index) => {
      const version = formatRevisionVersion(revision.id);
      return `
          <article class="version-item version-record">
            <div class="version-record-line">
              <strong>${escapeHtml(revision.changedBy)}</strong>
              <span>${index === 0 ? "current version" : "version"}</span>
              <code>${escapeHtml(version)}</code>
              <span>${escapeHtml(formatDateTime(revision.createdAt))}</span>
            </div>
            <div class="version-record-meta">
              <span>${revision.entries.length} keys</span>
              <span>${escapeHtml(revision.changeReason)}</span>
            </div>
          </article>
        `;
    })
    .join("");
}

function renderConfigVersionLabel() {
  const label = $("#configVersionLabel");
  if (!label) return;

  const project = activeProject();
  if (!project) {
    label.textContent = "No project selected";
    return;
  }

  const current = state.configHistory[0];
  if (!current) {
    label.textContent = `${state.activeEnvironment} · No saved version yet`;
    return;
  }

  const version = formatRevisionVersion(current.id);
  label.textContent = `${state.activeEnvironment} · Version ${version} · ${formatDateTime(current.createdAt)} · ${current.entries.length} keys`;
}

export function renderTemplateModal() {
  $("#templateModal").classList.toggle("hidden", !state.templateModalOpen);
  if (!state.templateModalOpen) return;

  const template = activeTemplate();
  const project = activeProject();
  const targetName = state.templatePickerActive
    ? state.projectDraft?.name || "New Project"
    : project?.name || "No project selected";
  if (!template) return;

  $("#templateModalTitle").textContent = template.name;
  $("#templateModalMeta").innerHTML = `
    <span>${escapeHtml(template.format)}</span>
    <span>${escapeHtml(targetName)}</span>
    <span>${escapeHtml(state.activeEnvironment)}</span>
  `;
  $("#confirmApplyTemplate").textContent = state.templatePickerActive
    ? "Use Template"
    : "Extract Config";
  $("#templateVariableList").innerHTML =
    template.variables
      .map(
        (variable) => `
      <label class="template-variable-field">
        <span>${escapeHtml(variable.name)}</span>
        <input
          type="${variable.isSensitive ? "password" : "text"}"
          value="${escapeHtml(state.templateValues[variable.name] ?? variable.defaultValue ?? "")}"
          placeholder="${escapeHtml(variable.description || variable.name)}"
          data-template-variable="${escapeHtml(variable.name)}"
          ${variable.required ? "required" : ""}
        />
      </label>
    `,
      )
      .join("") ||
    '<p class="project-meta">This template has no variables.</p>';
  $("#templateApplyFormat").value =
    state.templateApplyFormat || template.format || "yaml";
  $("#templateRenderedPreview").textContent = renderTemplateBody(template);
}

export function renderTemplateCreateModal() {
  $("#templateCreateModal").classList.toggle(
    "hidden",
    !state.templateCreateModalOpen,
  );
}

function activeTemplate() {
  return state.templates.find(
    (template) => template.id === state.activeTemplateId,
  );
}

function renderTemplateBody(template) {
  return (template?.body || "").replace(/\$\{([A-Z0-9_]+)\}/g, (_, name) => {
    const value =
      state.templateValues[name] ??
      template.variables.find((variable) => variable.name === name)
        ?.defaultValue ??
      "";
    return value;
  });
}

export function renderExportModal() {
  $("#exportModal").classList.toggle("hidden", !state.exportModalOpen);
  if (!state.exportModalOpen) return;

  const project = activeProject();
  $("#exportModalMeta").innerHTML = project
    ? `<span>${escapeHtml(project.name)}</span><span>${escapeHtml(state.activeEnvironment)}</span>`
    : "";
  $("#exportFormat").value =
    state.exportFormat || project?.defaultFormat || "yaml";
}

export function renderReviewModal() {
  $("#reviewModal").classList.toggle("hidden", !state.reviewModalOpen);
  if (!state.reviewModalOpen) return;

  const project = activeProject();
  const count = state.pendingReviewChanges.length;
  $("#reviewModalTitle").textContent =
    `Review ${count} ${count === 1 ? "Change" : "Changes"}`;
  $("#reviewModalMeta").innerHTML = project
    ? `<span>${escapeHtml(project.name)}</span><span>${escapeHtml(state.activeEnvironment)}</span>`
    : "";
  $("#reviewReason").value ||= project
    ? `Review ${state.activeEnvironment} config changes for ${project.name}`
    : "Review config changes";
  $("#reviewChangeList").innerHTML =
    state.pendingReviewChanges.map(renderReviewChange).join("") ||
    '<p class="project-meta">No pending changes.</p>';
}

function renderReviewChange(change) {
  const current = state.configs.find((config) => config.id === change.configId);
  const baseline = state.configBaseline.get(change.configId);
  const beforeKey = baseline?.key || "(new key)";
  const afterKey = current?.key || change.key;
  const beforeValue = baseline?.value ?? "";
  const afterValue = current?.value ?? "";
  const keyChanged = beforeKey !== afterKey;
  const valueChanged = beforeValue !== afterValue;

  return `
    <article class="review-change-item">
      <div>
        <strong>${escapeHtml(afterKey)}</strong>
        <p class="project-meta">
          <span>${keyChanged ? `key: ${escapeHtml(beforeKey)} -> ${escapeHtml(afterKey)}` : "key unchanged"}</span>
          <span>${valueChanged ? "value changed" : "value unchanged"}</span>
        </p>
      </div>
      <div class="review-value-pair">
        <code>${escapeHtml(beforeValue || "(empty)")}</code>
        <code>${escapeHtml(afterValue || "(empty)")}</code>
      </div>
    </article>
  `;
}

function formatRevisionVersion(id) {
  if (!id) return "current";
  return String(id)
    .replace(/^rev-/, "")
    .slice(0, 7);
}

export function renderImportPreview() {
  $("#importPreviewModal").classList.toggle("hidden", !state.importPreviewOpen);
  if (!state.importPreviewOpen) return;

  const preview = state.importPreview;
  $("#applyImportConfig").disabled = state.importApplying || !preview;
  if (!preview) {
    $("#importPreviewTitle").textContent = "Extracted Config";
    $("#importPreviewMeta").textContent = "";
    $("#importPreviewSummary").innerHTML =
      '<p class="project-meta">No extracted config yet.</p>';
    $("#importPreviewList").innerHTML = "";
    return;
  }

  $("#importPreviewTitle").textContent = `Extracted ${preview.fileName}`;
  $("#importPreviewMeta").innerHTML = `
    <span>${escapeHtml(preview.environment)}</span>
    <span>${escapeHtml(preview.format)}</span>
    <span>${preview.entryCount} keys</span>
  `;
  $("#importPreviewSummary").innerHTML = `
    <div class="history-previous current">
      <span>Created</span>
      <code>${preview.created}</code>
    </div>
    <div class="history-previous">
      <span>Updated</span>
      <code>${preview.updated}</code>
    </div>
    <div class="history-previous">
      <span>Unchanged</span>
      <code>${preview.unchanged}</code>
    </div>
  `;
  $("#importPreviewList").innerHTML = preview.entries
    .map(
      (entry) => `
        <article class="revision-entry import-preview-entry">
          <span>${escapeHtml(entry.key)}</span>
          <code>${escapeHtml(entry.value)}</code>
        </article>
      `,
    )
    .join("");
}

export function renderAll() {
  renderUser();
  renderNav();
  renderDashboard();
  renderProjects();
  renderTemplates();
  renderProjectTemplateOptions();
  renderTemplateModal();
  renderTemplateCreateModal();
  renderConfigRows();
  renderReviewDock();
  renderExportModal();
  renderReviewModal();
  renderRequests();
  renderVersionHistory();
  renderImportPreview();
}
