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

function isSystemView() {
  return state.user?.role === "system_admin";
}

function isReadOnlyView() {
  return state.user?.role === "viewer";
}

function isWorkspaceView() {
  return !isSystemView() && !isReadOnlyView();
}

function userDisplayName() {
  return state.user?.name || "";
}

function scopedProjectIds() {
  if (!isWorkspaceView()) return new Set(state.projects.map((project) => project.id));

  const actor = userDisplayName();
  const ids = new Set(state.projects.map((project) => project.id));
  state.projects.forEach((project) => {
  });
  state.requests.forEach((request) => {
    if (request.requester === actor || request.reviewer === actor) {
      ids.add(request.projectId);
    }
  });
  state.configs.forEach((config) => {
    if (config.updatedBy === actor || config.updated === actor) {
      ids.add(config.projectId);
    }
  });
  state.groups.forEach((group) => {
    (group.projects || group.managedProjects || []).forEach((project) => {
      ids.add(typeof project === "string" ? project : project.id);
    });
  });
  return ids;
}

function projectInScope(project, scope = scopedProjectIds()) {
  return scope.has(project.id);
}

function filteredProjects() {
  const term = searchTerm();
  const scope = scopedProjectIds();
  return state.projects.filter(
    (project) =>
      projectInScope(project, scope) &&
      includesSearch(
        [
          project.name,
          project.repoUrl,
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
        template.isCustom ? "project" : "global",
        ...template.keys,
      ],
      term,
    ),
  );
}

function filteredSharedConfigs() {
  const term = searchTerm();
  return state.sharedConfigs.filter((item) =>
    includesSearch(
      [
        item.name,
        item.description,
        item.scope,
        item.scopeName,
        item.format,
        item.updatedBy,
        ...item.keys,
      ],
      term,
    ),
  );
}

function filteredRequests() {
  if (isReadOnlyView()) return [];
  const term = searchTerm();
  const scope = scopedProjectIds();
  return state.requests.filter(
    (request) =>
      scope.has(request.projectId) &&
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
  const file = configFileForEntry(config, state);
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
  const visibleItems = navItems.filter(
    (item) => !(isReadOnlyView() && item.id === "requests"),
  );
  $("#navList").innerHTML = visibleItems
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
  $("#userName").textContent = state.user?.name || "Not signed in";
  renderUserMenu();
}

export function renderUserMenu() {
  const button = $("#userMenuButton");
  const menu = $("#userMenu");
  if (!button || !menu) return;

  button.setAttribute("aria-expanded", state.userMenuOpen ? "true" : "false");
  menu.innerHTML = `
    <button type="button" role="menuitem" data-user-menu-action="groups">Group</button>
    <button type="button" role="menuitem" data-user-menu-action="logout">Log out</button>
  `;
}


function pendingReviewRequests() {
  return filteredRequests().filter((request) => request.status === "pending");
}

function prodPendingRequests(requests) {
  return requests.filter((request) => request.environment === "prod");
}

function configsInScope() {
  const scope = scopedProjectIds();
  return state.configs.filter((config) => scope.has(config.projectId));
}

function sensitiveConfigs() {
  return configsInScope().filter((config) => config.isSensitive);
}

function sensitiveRevealKey(config) {
  return `${config.projectId}:${config.environment}:${config.key}`;
}

function keyLooksSensitive(key) {
  return /(password|secret|token|credential|database\.url|db\.url|database.*url|db.*url)/i.test(key);
}

function exposedSensitiveConfigs() {
  return configsInScope().filter((config) => {
    if (config.environment !== "prod") return false;
    const revealedSensitiveValue =
      config.isSensitive && state.revealedKeys.has(sensitiveRevealKey(config));
    const untaggedSensitiveKey = !config.isSensitive && keyLooksSensitive(config.key);
    return revealedSensitiveValue || untaggedSensitiveKey;
  });
}

function templatedProjectCount(projects = filteredProjects()) {
  return projects.filter((project) => project.templateId).length;
}

function myRecentChangeCount() {
  const actor = userDisplayName();
  return state.configHistory.filter((revision) => revision.changedBy === actor).length;
}

function dashboardButtonAttrs(attrs) {
  return Object.entries(attrs)
    .filter(([, value]) => value !== undefined && value !== "")
    .map(([key, value]) => `${key}="${escapeHtml(value)}"`)
    .join(" ");
}

function renderAttentionRow({ title, meta, tone = "neutral", status, attrs }) {
  return `
    <button class="dashboard-row attention-row ${tone}" type="button" ${dashboardButtonAttrs(attrs)}>
      <span class="attention-marker" aria-hidden="true"></span>
      <div>
        <h3>${escapeHtml(title)}</h3>
        <p class="project-meta">${meta.map((item) => `<span>${escapeHtml(item)}</span>`).join("")}</p>
      </div>
      <span class="status-pill ${tone === "danger" ? "danger" : tone === "warning" ? "warning" : "neutral"}">${escapeHtml(status)}</span>
    </button>
  `;
}

export function renderStats() {
  const projects = filteredProjects();
  const pending = pendingReviewRequests();
  const prodPending = prodPendingRequests(pending).length;
  const sensitiveCount = sensitiveConfigs().length;
  const exposedSensitiveCount = exposedSensitiveConfigs().length;
  const templatedCount = templatedProjectCount(projects);
  const stats = [];

  if (!isReadOnlyView()) {
    stats.push({
      label: isSystemView() ? "Global Pending Reviews" : "My Pending Reviews",
      value: pending.length,
      change: prodPending ? `${prodPending} prod guarded` : "Queue clear",
      tone: pending.length ? "warning" : "success",
    });
  }

  stats.push({
    label: "Sensitive Keys",
    value: sensitiveCount,
    change: exposedSensitiveCount
      ? `${exposedSensitiveCount} exposed in prod`
      : "All masked",
    tone: exposedSensitiveCount && !isReadOnlyView() ? "danger" : "info",
  });

  if (isSystemView()) {
    stats.push({
      label: "Template Coverage",
      value: `${templatedCount}/${projects.length || 0}`,
      change: projects.length === templatedCount ? "Standardized" : "Needs review",
      tone: projects.length === templatedCount ? "success" : "neutral",
    });
  } else if (isReadOnlyView()) {
    stats.push({
      label: "Projects",
      value: projects.length,
      change: "Read only",
      tone: "neutral",
    });
    stats.push({
      label: "Recent Changes",
      value: state.configHistory.length,
      change: "Current project",
      tone: "neutral",
    });
  } else {
    stats.push({
      label: "My Projects",
      value: projects.length,
      change: "Workspace scope",
      tone: "neutral",
    });
    stats.push({
      label: "My Recent Changes",
      value: myRecentChangeCount(),
      change: "Current project",
      tone: "neutral",
    });
  }

  $("#statsGrid").innerHTML = stats
    .map(
      (stat) => `
        <article class="stat-card ${stat.tone}">
          <p>${escapeHtml(stat.label)}</p>
          <strong>${escapeHtml(stat.value)}</strong>
          <span class="metric-change">${escapeHtml(stat.change)}</span>
        </article>
      `,
    )
    .join("");
}

export function renderDashboard() {
  renderStats();
  const attentionPanel = document.querySelector(".attention-panel");
  if (attentionPanel) {
    attentionPanel.classList.toggle("hidden", isReadOnlyView());
  }
  renderDashboardAttention();
  renderDashboardActivity();
  renderDashboardCoverage();
}

function renderDashboardAttention() {
  if (isReadOnlyView()) {
    const attention = $("#dashboardAttention");
    if (attention) attention.innerHTML = "";
    return;
  }
  const project = filteredProjects().find((item) => item.id === activeProject()?.id) || filteredProjects()[0];
  const pending = pendingReviewRequests();
  const exposedSensitive = exposedSensitiveConfigs();
  const rows = pending.slice(0, 3).map((request) =>
    renderAttentionRow({
      title: request.reason || `Review ${request.id.slice(0, 8)}`,
      meta: [request.projectName, request.environment, request.requester],
      tone: request.environment === "prod" ? "warning" : "neutral",
      status: request.environment === "prod" ? "prod" : "review",
      attrs: { "data-jump": "requests" },
    }),
  );

  if (project && state.pendingReviewChanges.length) {
    rows.unshift(
      renderAttentionRow({
        title: "Unsaved review draft",
        meta: [project.name, state.activeEnvironment, `${state.pendingReviewChanges.length} local changes`],
        tone: state.activeEnvironment === "prod" ? "warning" : "neutral",
        status: "draft",
        attrs: { "data-open-config": project.id },
      }),
    );
  }

  if (project && exposedSensitive.length) {
    rows.push(
      renderAttentionRow({
        title: `${exposedSensitive.length} sensitive string${exposedSensitive.length === 1 ? "" : "s"} exposed in Prod`,
        meta: [project.name, "prod", "unmasked or unlabeled secret"],
        tone: "danger",
        status: "exposed",
        attrs: { "data-open-config": project.id },
      }),
    );
  }

  $("#dashboardAttention").innerHTML = rows.join("") || `
    <article class="dashboard-empty">
      <h3>No urgent config actions</h3>
      <p class="project-meta"><span>Reviews clear</span><span>Sensitive values masked</span></p>
    </article>
  `;
}

function renderDashboardActivity() {
  const project = filteredProjects().find((item) => item.id === activeProject()?.id);
  const revisions = project ? state.configHistory : [];
  const action = $("#dashboardHistoryAction");
  if (action) {
    action.disabled = !project || !revisions.length;
  }

  $("#dashboardActivity").innerHTML = revisions.length
    ? revisions
        .slice(0, 5)
        .map(
          (revision, index) => `
            <article class="dashboard-row activity-row">
              <div class="activity-version">
                <span>${index === 0 ? "Current" : "Version"}</span>
                <code>${escapeHtml(formatRevisionVersion(revision.id))}</code>
              </div>
              <div>
                <h3>${escapeHtml(revision.changeReason || "Config revision")}</h3>
                <p class="project-meta">
                  <span>${escapeHtml(revision.changedBy)}</span>
                  <span>${revision.entries.length} keys</span>
                  <span>${escapeHtml(formatDateTime(revision.createdAt))}</span>
                </p>
              </div>
            </article>
          `,
        )
        .join("")
    : `
      <article class="dashboard-empty">
        <h3>No revision history yet</h3>
        <p class="project-meta"><span>${escapeHtml(project?.name || "Select a project")}</span><span>${escapeHtml(state.activeEnvironment)}</span></p>
      </article>
    `;
}

function renderDashboardCoverage() {
  const projects = filteredProjects().slice(0, 4);
  const scopedProjects = filteredProjects();
  const sharedTemplates = state.templates.filter((template) => !template.isCustom).length;
  const personalTemplates = state.templates.filter((template) => template.isCustom).length;
  const templatedCount = templatedProjectCount(scopedProjects);
  const untemplatedCount = Math.max(scopedProjects.length - templatedCount, 0);
  const pending = pendingReviewRequests().length;

  const summary = isSystemView()
    ? `
      <div class="coverage-summary">
        <div>
          <span>Projects</span>
          <strong>${scopedProjects.length}</strong>
        </div>
        <div>
          <span>Library</span>
          <strong>${sharedTemplates}+${personalTemplates}</strong>
        </div>
        <div>
          <span>Groups</span>
          <strong>${state.groups.length}</strong>
        </div>
        <div>
          <span>Unstandardized</span>
          <strong>${untemplatedCount}</strong>
        </div>
      </div>
    `
    : `
      <div class="coverage-summary">
        <div>
          <span>${isReadOnlyView() ? "Readable Projects" : "My Projects"}</span>
          <strong>${scopedProjects.length}</strong>
        </div>
        <div>
          <span>${isReadOnlyView() ? "Visible Keys" : "Pending Reviews"}</span>
          <strong>${isReadOnlyView() ? configsInScope().length : pending}</strong>
        </div>
        <div>
          <span>Groups</span>
          <strong>${state.groups.length}</strong>
        </div>
        <div>
          <span>Recent Changes</span>
          <strong>${isWorkspaceView() ? myRecentChangeCount() : state.configHistory.length}</strong>
        </div>
      </div>
    `;

  const projectRows = projects
    .map((project) => {
      const hasProd = project.environments.includes("prod");
      const templateName = projectTemplateName(project.templateId);
      return `
        <button class="dashboard-row coverage-row" type="button" data-open-config="${escapeHtml(project.id)}">
          <div>
            <h3>${escapeHtml(project.name)}</h3>
            <p class="project-meta">
              <span>${project.configCount} keys</span>
              <span>${escapeHtml(templateName || "No template")}</span>
            </p>
          </div>
          <div class="environment-strip compact-env-strip">
            ${project.environments.map((environment) => `<span>${escapeHtml(environment)}</span>`).join("")}
          </div>
          <span class="status-pill ${hasProd ? "success" : "warning"}">${hasProd ? "prod" : "no prod"}</span>
        </button>
      `;
    })
    .join("") || '<p class="project-meta">No matching projects.</p>';

  $("#dashboardCoverage").innerHTML = `${summary}<div class="coverage-projects">${projectRows}</div>`;
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
              <span>${project.configCount} config keys</span>
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
    const canCreateSharedConfig = state.libraryTab === "shared-config" && isSystemView();
    action.textContent = canCreateSharedConfig ? "New Global Shared Config" : action.textContent;
    action.className = canCreateSharedConfig ? "primary-action" : action.className;
    action.classList.toggle("hidden", state.libraryTab === "shared-config" && !isSystemView());
  }

  const isSharedTab = state.libraryTab === "shared-config";
  const items = isSharedTab ? filteredSharedConfigs() : filteredTemplates();
  $("#templatesGrid").innerHTML = `
    <div class="library-toolbar">
      <div class="segmented-control library-tabs" role="tablist">
        <button class="${!isSharedTab ? "active" : ""}" type="button" data-library-tab="templates">Templates</button>
        <button class="${isSharedTab ? "active" : ""}" type="button" data-library-tab="shared-config">Shared Config</button>
      </div>
    </div>
    <div class="library-grid">
      ${items.map(isSharedTab ? renderSharedConfigCard : renderTemplateCard).join("") || `<p class="project-meta">No matching ${isSharedTab ? "shared configs" : "templates"}.</p>`}
    </div>
  `;
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
  const scope = libraryScopeLabel(template);
  return `
    <${tag} class="template-card ${canPick ? "template-card-button" : ""}"${typeAttr}${pickAttr}>
      <div class="card-top">
        <div class="card-title">
          <h3>${escapeHtml(template.name)}</h3>
          <p>${escapeHtml(template.description || "Reusable configuration template")}</p>
        </div>
        <div class="template-badges">
          <span class="library-pill type">Template</span>
          ${renderLibraryScopePill(scope)}
        </div>
      </div>
      <pre class="template-body-preview">${escapeHtml(template.body || template.entries?.map((entry) => `${entry.key}=${entry.defaultValue}`).join("\n") || "")}</pre>
      <div class="template-list">
        ${renderTemplateKeys(template)}
      </div>
    </${tag}>
  `;
}

function renderSharedConfigCard(item) {
  const scope = libraryScopeLabel(item);
  return `
    <article class="template-card shared-config-card">
      <div class="card-top">
        <div class="card-title">
          <h3>${escapeHtml(item.name)}</h3>
          <p>${escapeHtml(item.description || "Config inherited by projects")}</p>
        </div>
        <div class="template-badges shared-card-tools">
          ${isSystemView() && scope === "Global" ? `<button class="icon-button small-icon-button" type="button" aria-label="Edit ${escapeHtml(item.name)}" title="Edit" data-edit-shared-config="${escapeHtml(item.id)}"><span aria-hidden="true">✎</span></button>` : ""}
          <span class="library-pill type">Shared Config</span>
          ${renderLibraryScopePill(scope)}
        </div>
      </div>
      <div class="shared-config-summary">
        <span>${item.entries.length} keys</span>
        <span>${escapeHtml(item.scopeName || scope)}</span>
        <span>${item.inheritedBy || 0} projects</span>
      </div>
      <div class="template-list">
        ${renderTemplateKeys(item)}
      </div>
      <p class="project-meta"><span>Updated by ${escapeHtml(item.updatedBy || "System")}</span></p>
      <div class="template-card-actions">
        ${isSystemView() && scope === "Global" ? `<button class="ghost-action danger-action" type="button" data-delete-shared-config="${escapeHtml(item.id)}">Delete</button>` : ""}
      </div>
    </article>
  `;
}

function libraryScopeLabel(item) {
  const rawScope = String(item.scope || item.scopeType || "").toLowerCase();
  if (rawScope === "global") return "Global";
  if (rawScope === "group") return "Group";
  if (rawScope === "project") return "Project";
  return item.isCustom ? "Project" : "Global";
}

function renderLibraryScopePill(scope) {
  return `<span class="library-pill scope ${escapeHtml(scope.toLowerCase())}">${escapeHtml(scope)}</span>`;
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
  const createForm = state.configFileCreateOpen ? renderConfigFileCreateForm() : "";
  const files = configFilesForEntries(configEntriesForFileList(), state)
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
  $("#configProjectList").innerHTML = `${createForm}${files}`;
}

function renderConfigFileCreateForm() {
  const sourceType = state.configFileSourceType || "blank";
  return `
    <form id="configFileCreateForm" class="config-file-create-form">
      <input id="newConfigFileName" type="text" autocomplete="off" placeholder="database.yaml" value="${escapeHtml(state.configFileDraftName || "")}" required />
      <select id="configFileSourceType" aria-label="Config file source">
        <option value="blank" ${sourceType === "blank" ? "selected" : ""}>Blank</option>
        <option value="template" ${sourceType === "template" ? "selected" : ""}>Template</option>
        <option value="shared-config" ${sourceType === "shared-config" ? "selected" : ""}>Shared Config</option>
      </select>
      ${renderConfigFileSourcePicker(sourceType)}
    </form>
  `;
}

function renderConfigFileSourcePicker(sourceType) {
  const items = sourceType === "template" ? state.templates : sourceType === "shared-config" ? state.sharedConfigs : [];
  if (!items.length) return "";
  return `
    <select id="configFileSourceId" aria-label="Source item">
      <option value="">No source selected</option>
      ${items
        .map((item) => `<option value="${escapeHtml(item.id)}" ${item.id === state.configFileSourceId ? "selected" : ""}>${escapeHtml(item.name || item.id)}</option>`)
        .join("")}
    </select>
  `;
}

function configEntriesForFileList() {
  if (state.configMode !== "compare") return state.configs;
  return [
    ...(state.compareConfigs[state.compareSourceEnv] || []),
    ...(state.compareConfigs[state.compareTargetEnv] || []),
  ];
}

export function renderEnvironmentTabs() {
  const project = activeProject();
  const modeTabs = $("#configModeTabs");
  const environmentTabs = $("#environmentTabs");
  const compareControls = $("#compareControls");
  if (!project) {
    if (modeTabs) modeTabs.innerHTML = "";
    if (environmentTabs) environmentTabs.innerHTML = "";
    if (compareControls) compareControls.innerHTML = "";
    return;
  }
  if (!project.environments.includes(state.activeEnvironment)) {
    state.activeEnvironment = project.environments[0];
  }
  ensureCompareEnvironments(project.environments);

  if (modeTabs) {
    modeTabs.innerHTML = ["view", "compare"]
      .map(
        (mode) => `
          <button class="${state.configMode === mode ? "active" : ""}" type="button" data-config-mode="${mode}">
            ${mode === "view" ? "View" : "Compare"}
          </button>
        `,
      )
      .join("");
  }

  if (state.configMode === "compare") {
    environmentTabs.classList.add("hidden");
    compareControls.classList.remove("hidden");
    compareControls.innerHTML = `
      <select data-compare-env="source" aria-label="Source environment">
        ${renderEnvironmentOptions(project.environments, state.compareSourceEnv)}
      </select>
      <span>vs</span>
      <select data-compare-env="target" aria-label="Target environment">
        ${renderEnvironmentOptions(project.environments, state.compareTargetEnv)}
      </select>
    `;
    return;
  }

  compareControls.classList.add("hidden");
  compareControls.innerHTML = "";
  environmentTabs.classList.remove("hidden");
  environmentTabs.innerHTML = project.environments
    .map(
      (environment) => `
        <button class="${environment === state.activeEnvironment ? "active" : ""}" type="button" data-env="${environment}">
          ${environment}
        </button>
      `,
    )
    .join("");
}

function ensureCompareEnvironments(environments) {
  if (!environments.length) return;
  if (!environments.includes(state.compareSourceEnv)) {
    state.compareSourceEnv = environments[0];
  }
  if (!environments.includes(state.compareTargetEnv)) {
    state.compareTargetEnv = environments.find((env) => env !== state.compareSourceEnv) || environments[0];
  }
}

function renderEnvironmentOptions(environments, selected) {
  return environments
    .map(
      (environment) => `<option value="${escapeHtml(environment)}" ${environment === selected ? "selected" : ""}>${escapeHtml(environment)}</option>`,
    )
    .join("");
}

export function renderProjectTemplateOptions() {
  const summary = $("#projectTemplateSelection");
  const clearButton = $("#clearProjectTemplate");
  const groupSelect = $("#projectGroup");
  if (!summary) return;

  const selection = state.projectTemplateSelection;
  summary.textContent = selection
    ? `${selection.templateName} · ${selection.outputFormat}`
    : "No template selected.";
  if (clearButton) {
    clearButton.classList.toggle("hidden", !selection);
  }
  if (groupSelect) {
    const current = groupSelect.value || state.projectDraft?.groupId || "";
    groupSelect.innerHTML = [
      `<option value="" disabled ${current ? "" : "selected"}>Select group</option>`,
      ...state.groups.map(
        (group) => `<option value="${escapeHtml(group.id)}" ${group.id === current ? "selected" : ""}>${escapeHtml(group.name || group.id)}</option>`,
      ),
    ].join("");
    if (current && state.groups.some((group) => group.id === current)) {
      groupSelect.value = current;
    } else if (!current && state.groups.length === 1) {
      groupSelect.value = state.groups[0].id;
    }
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

  if (state.configMode === "compare") {
    renderCompareConfigRows();
    renderReviewDock();
    return;
  }

  renderViewTableHead();
  const rows = configsForActiveFile(
    state.configs,
    state.activeConfigFile,
    state,
  ).filter(matchesConfigSearch);

  $("#configRows").innerHTML =
    rows.map(renderConfigRowMarkup).join("") ||
    `<tr><td colspan="3" class="value-cell">No config keys match this view.</td></tr>`;
  renderReviewDock();
}

function renderViewTableHead() {
  const tableHead = $("#configTableHead");
  if (!tableHead) return;
  tableHead.innerHTML = `
    <tr>
      <th>Key</th>
      <th>Value</th>
      <th>Updated</th>
    </tr>
  `;
}

function renderCompareTableHead() {
  const tableHead = $("#configTableHead");
  if (!tableHead) return;
  tableHead.innerHTML = `
    <tr>
      <th>Key</th>
      <th>${escapeHtml(state.compareSourceEnv)} Value</th>
      <th>${escapeHtml(state.compareTargetEnv)} Value</th>
      <th>Status</th>
    </tr>
  `;
}

function renderCompareConfigRows() {
  renderCompareTableHead();
  if (state.compareLoading) {
    $("#configRows").innerHTML = `<tr><td colspan="4" class="value-cell">Loading environment comparison.</td></tr>`;
    return;
  }

  const rows = compareRowsForActiveFile().filter((row) => matchesCompareSearch(row));
  $("#configRows").innerHTML =
    rows.map(renderCompareRowMarkup).join("") ||
    `<tr><td colspan="4" class="value-cell">No config keys match this comparison.</td></tr>`;
}

function compareRowsForActiveFile() {
  const sourceEntries = configsForActiveFile(
    state.compareConfigs[state.compareSourceEnv] || [],
    state.activeConfigFile,
    state,
  );
  const targetEntries = configsForActiveFile(
    state.compareConfigs[state.compareTargetEnv] || [],
    state.activeConfigFile,
    state,
  );
  const sourceByKey = new Map(sourceEntries.map((entry) => [entry.key, entry]));
  const targetByKey = new Map(targetEntries.map((entry) => [entry.key, entry]));
  return Array.from(new Set([...sourceByKey.keys(), ...targetByKey.keys()]))
    .sort((a, b) => a.localeCompare(b))
    .map((key) => buildCompareRow(key, sourceByKey.get(key), targetByKey.get(key)));
}

function buildCompareRow(key, source, target) {
  if (!source || !target) {
    return { key, source, target, status: "Missing", tone: "missing" };
  }
  if (String(source.value) !== String(target.value)) {
    return { key, source, target, status: "Modified", tone: "modified" };
  }
  return { key, source, target, status: "Same", tone: "same" };
}

function matchesCompareSearch(row) {
  const local = state.configSearch.trim().toLowerCase();
  const global = searchTerm();
  const values = [
    row.key,
    row.source?.value,
    row.target?.value,
    row.source?.updatedBy,
    row.target?.updatedBy,
    row.status,
  ];
  return includesSearch(values, local) && includesSearch(values, global);
}

function renderCompareRowMarkup(row) {
  return `
    <tr class="compare-row ${row.tone === "modified" ? "diff-modified" : ""}" data-compare-row="${escapeHtml(row.key)}">
      <td class="key-cell">${escapeHtml(row.key)}</td>
      ${renderCompareValueCell(row.source, row.tone === "missing" && !row.source)}
      ${renderCompareValueCell(row.target, row.tone === "missing" && !row.target)}
      <td>${renderCompareStatus(row)}</td>
    </tr>
  `;
}

function renderCompareValueCell(entry, missing) {
  if (missing) {
    return `<td class="value-cell missing-value"><span>-</span><span class="status-pill danger">Missing</span></td>`;
  }
  if (!entry) return `<td class="value-cell missing-value"><span>-</span></td>`;
  const value = entry.isSensitive ? "******" : entry.value;
  return `<td class="value-cell">${escapeHtml(value)}</td>`;
}

function renderCompareStatus(row) {
  if (row.status === "Missing") {
    return `<span class="status-pill danger">Missing</span>`;
  }
  if (row.status === "Modified") {
    return `<span class="diff-status"><span class="diff-dot" aria-hidden="true"></span>Modified</span>`;
  }
  return `<span class="status-pill neutral">Same</span>`;
}

export function renderConfigRow(configId) {
  const config = state.configs.find((entry) => entry.id === configId);
  const row = Array.from(document.querySelectorAll("[data-config-row]")).find(
    (element) => element.dataset.configRow === configId,
  );

  if (
    !config ||
    !row ||
    !configsForActiveFile([config], state.activeConfigFile, state).length ||
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
    state.notifications.filter((notification) => !notification.read).length,
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



export function renderConfigCreateDrawer() {
  const drawer = $("#configCreateDrawer");
  if (!drawer) return;
  drawer.classList.toggle("hidden", !state.configCreateDrawerOpen);
  if (!state.configCreateDrawerOpen) return;

  const valueType = state.newConfigValueType || $("#newConfigValueType")?.value || "string";
  const typeSelect = $("#newConfigValueType");
  if (typeSelect) typeSelect.value = valueType;
  renderNewConfigEnvironmentValues(valueType);
}

function renderNewConfigEnvironmentValues(valueType) {
  const target = $("#newConfigEnvironmentValues");
  if (!target) return;
  const project = activeProject();
  const environments = project?.environments?.length ? project.environments : ["dev", "staging", "prod"];
  target.innerHTML = environments
    .map((environment) => renderNewConfigEnvironmentField(environment, valueType))
    .join("");
}

function renderNewConfigEnvironmentField(environment, valueType) {
  const id = `newConfigValue${environmentFieldSuffix(environment)}`;
  const label = escapeHtml(`${environment} value`);
  if (valueType === "boolean") {
    return `
      <label class="environment-value-field compact-control">
        <span>${label}</span>
        <select id="${escapeHtml(id)}" data-new-config-value="${escapeHtml(environment)}">
          <option value="true">true</option>
          <option value="false" selected>false</option>
        </select>
      </label>
    `;
  }
  if (["json", "yaml"].includes(valueType)) {
    return `
      <label class="environment-value-field">
        <span>${label}</span>
        <textarea id="${escapeHtml(id)}" data-new-config-value="${escapeHtml(environment)}" rows="4" spellcheck="false"></textarea>
      </label>
    `;
  }
  return `
    <label class="environment-value-field compact-control">
      <span>${label}</span>
      <input id="${escapeHtml(id)}" data-new-config-value="${escapeHtml(environment)}" type="text" autocomplete="off" />
    </label>
  `;
}

function environmentFieldSuffix(environment) {
  return String(environment || "")
    .split(/[^a-z0-9]+/i)
    .filter(Boolean)
    .map((part) => `${part.charAt(0).toUpperCase()}${part.slice(1)}`)
    .join("");
}

export function renderSharedConfigEditModal() {
  const modal = $("#sharedConfigEditModal");
  if (!modal) return;
  modal.classList.toggle("hidden", !state.sharedConfigEditModalOpen);
  if (!state.sharedConfigEditModalOpen) return;
  const item = state.sharedConfigs.find((config) => config.id === state.activeSharedConfigId);
  const affected = item?.inheritedBy || item?.affectedProjects?.length || 0;
  const prod = item?.prodEnvironmentCount || 0;
  const impact = $("#sharedConfigImpact");
  if (impact) {
    impact.innerHTML = `<strong>Impact Analysis</strong><span>This change will affect ${affected} projects and ${prod} production environments.</span>`;
  }
}

export function renderSharedConfigCreateModal() {
  const modal = $("#sharedConfigCreateModal");
  if (!modal) return;
  modal.classList.toggle("hidden", !state.sharedConfigCreateModalOpen);
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
  $("#exportFormat").value = state.exportFormat || "yaml";
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


function canCreateGroup() {
  return ["system_admin", "group_admin"].includes(state.user?.role);
}

function canEditGroupMembers() {
  return ["system_admin", "group_admin"].includes(state.user?.role);
}

function groupMembers() {
  return state.activeGroup?.members || [];
}

function userLabel(user) {
  return `${user.name || user.id}${user.email ? ` · ${user.email}` : ""}`;
}

function roleMode() {
  return state.user?.role === "system_admin" ? "system" : "user";
}

function groupProjects(group) {
  return group?.projects || group?.managedProjects || [];
}

function renderScopeChips(items, emptyLabel) {
  if (!items?.length) return `<p class="project-meta">${emptyLabel}</p>`;
  return `
    <div class="group-chip-list">
      ${items
        .map((item) => {
          const label = typeof item === "string" ? item : item.name || item.id;
          return `<span>${escapeHtml(label || "Untitled")}</span>`;
        })
        .join("")}
    </div>
  `;
}

function groupTabButton(tab, label) {
  return `<button class="group-tab ${state.groupDetailTab === tab ? "active" : ""}" type="button" data-group-tab="${tab}">${label}</button>`;
}

function itemLabel(item) {
  return typeof item === "string" ? item : item.name || item.id || "Untitled";
}

function renderPlainList(items, emptyLabel) {
  if (!items?.length) return `<p class="group-empty-state">${emptyLabel}</p>`;
  return `
    <div class="group-plain-list">
      ${items
        .map(
          (item) => `
            <article class="group-plain-row">
              <strong>${escapeHtml(itemLabel(item))}</strong>
              ${typeof item === "string" || !item.email ? "" : `<p>${escapeHtml(item.email)}</p>`}
            </article>
          `,
        )
        .join("")}
    </div>
  `;
}

function groupRoleLabel(role) {
  return role === "group_admin" ? "Group Admin" : "Member";
}

function renderGroupRoleControl(member, editAllowed) {
  const id = member.id || member.userId;
  const role = member.groupRole === "group_admin" ? "group_admin" : "member";
  if (!editAllowed) return `<span class="group-role-text">${groupRoleLabel(role)}</span>`;
  
  const isOpen = state.groupRoleMenuUserId === id;

  return `
    <div class="group-role-dropdown-container" style="position: relative;">
      <button type="button" class="group-role-button" data-toggle-role-menu="${escapeHtml(id)}">
        <span>${groupRoleLabel(role)}</span>
        <svg viewBox="0 0 24 24" width="14" height="14" stroke="#86868b" stroke-width="2.5" fill="none" stroke-linecap="round" stroke-linejoin="round" class="chevron"><polyline points="6 9 12 15 18 9"></polyline></svg>
      </button>
      ${isOpen ? `
        <div class="apple-menu">
          <button type="button" class="apple-menu-item ${role === 'member' ? 'active' : ''}" data-set-role="member" data-user-id="${escapeHtml(id)}">
            <span class="apple-menu-icon" aria-hidden="true">${role === 'member' ? '<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>' : ''}</span>
            <span>Member</span>
          </button>
          <div class="apple-menu-divider"></div>
          <button type="button" class="apple-menu-item ${role === 'group_admin' ? 'active' : ''}" data-set-role="group_admin" data-user-id="${escapeHtml(id)}">
            <span class="apple-menu-icon" aria-hidden="true">${role === 'group_admin' ? '<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>' : ''}</span>
            <span>Group Admin</span>
          </button>
        </div>
      ` : ""}
    </div>
  `;
}

function renderMemberRows(members, editAllowed) {
  if (!members.length) return `<p class="group-empty-state">No members in this group.</p>`;
  return `
    <div class="group-plain-list group-member-list">
      ${members
        .map((member) => {
          const id = member.id || member.userId;
          const isCurrentUser = state.user?.id === id;
          return `
            <article class="group-member-row ${isCurrentUser ? "current-user-row" : ""}">
              <div class="member-avatar">${escapeHtml(initials(member.name || id || "--"))}</div>
              <div>
                <strong>${escapeHtml(member.name || id || "Unknown user")}</strong>
                <p class="project-meta">
                  ${isCurrentUser ? `<span class="user-icon" aria-label="You"><svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/></svg></span>` : member.email ? `<span>${escapeHtml(member.email)}</span>` : ""}
                </p>
              </div>
              <div class="group-member-controls">
                ${renderGroupRoleControl(member, editAllowed)}
                ${editAllowed ? `<button class="member-remove-button" type="button" data-remove-group-member="${escapeHtml(id)}" aria-label="Remove ${escapeHtml(member.name || id || "member")}"><svg aria-hidden="true" viewBox="0 0 24 24"><path d="M9 3h6l1 2h4v2H4V5h4l1-2Zm-2 6h10l-.7 11H7.7L7 9Zm3 2v7h2v-7h-2Zm4 0v7h2v-7h-2Z"></path></svg></button>` : ""}
              </div>
            </article>
          `;
        })
        .join("")}
    </div>
  `;
}

function userMatchesSearch(user, term) {
  if (!term) return true;
  return [user.name, user.email, user.id, user.role].some((value) =>
    String(value || "")
      .toLowerCase()
      .includes(term),
  );
}

export function renderMemberPicker({
  users,
  selectedIds,
  search,
  disabledIds = new Set(),
  searchId,
  optionAttribute,
  removeAttribute,
  submitId = "",
  emptyLabel,
}) {
  const selectedSet = selectedIds instanceof Set ? selectedIds : new Set(selectedIds || []);
  const term = search.trim().toLowerCase();
  const visibleUsers = users.filter((user) => userMatchesSearch(user, term));
  const selectedUsers = users.filter((user) => selectedSet.has(user.id));
  const submitButton = submitId
    ? `<button id="${submitId}" class="primary-action" type="button" ${selectedUsers.length === 0 ? "disabled" : ""}>Add ${selectedUsers.length || ""} Member${selectedUsers.length === 1 ? "" : "s"}</button>`
    : "";

  return `
    <div class="member-picker">
      <div class="member-picker-grid">
        <div class="member-picker-list">
          <div class="member-picker-list-head">
            <div class="member-picker-search">
              <input id="${searchId}" type="search" value="${escapeHtml(search)}" placeholder="Search name or email" autocomplete="off" />
            </div>
          </div>
          <div class="member-picker-options">
            ${visibleUsers.length
              ? visibleUsers
                  .map((user) => {
                    const disabled = disabledIds.has(user.id);
                    const checked = selectedSet.has(user.id) || disabled;
                    return `
                      <label class="member-picker-option ${disabled ? "disabled" : ""}">
                        <input
                          type="checkbox"
                          ${optionAttribute}
                          value="${escapeHtml(user.id)}"
                          ${checked ? "checked" : ""}
                          ${disabled ? "disabled" : ""}
                        />
                        <span>
                          <strong>${escapeHtml(user.name || user.id || "Unknown user")}</strong>
                          <small>${escapeHtml(disabled ? "Already member" : user.email || user.id || "")}</small>
                        </span>
                      </label>
                    `;
                  })
                  .join("")
              : `<p class="project-meta">${emptyLabel}</p>`}
          </div>
        </div>
        <aside class="member-picker-selected">
          ${selectedUsers.length ? `
            <div class="member-picker-selected-head">
              <span>Selected</span>
              <strong>${selectedUsers.length}</strong>
            </div>
          ` : ""}
          <div class="selected-member-list">
            ${selectedUsers.length
              ? selectedUsers
                  .map(
                    (user) => `
                      <button class="selected-member-chip" type="button" ${removeAttribute}="${escapeHtml(user.id)}">
                        <span>${escapeHtml(user.name || user.id || "Unknown user")}</span>
                        <span class="remove-chip-icon" style="color: #a1a1a6;" aria-hidden="true">
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                        </span>
                      </button>
                    `,
                  )
                  .join("")
              : '<div class="empty-selection-placeholder" style="text-align: center; color: #86868b; margin-top: 40px;"><svg aria-hidden="true" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin-bottom: 12px; opacity: 0.3;"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg><br>No members selected</div>'}
          </div>
          <div class="member-picker-actions" style="margin-top: auto; display: flex; justify-content: flex-end; padding-top: 16px;">
            ${submitButton}
          </div>
        </aside>
      </div>
    </div>
  `;
}

export function renderGroupCreateMemberPicker() {
  const target = $("#groupCreateMemberPicker");
  if (!target) return;
  target.innerHTML = renderMemberPicker({
    users: state.users,
    selectedIds: state.groupCreateMemberSelection,
    search: state.groupCreateMemberSearch,
    searchId: "groupCreateMemberSearch",
    optionAttribute: "data-group-create-member-option",
    removeAttribute: "data-remove-selected-create-member",
    emptyLabel: "No users found.",
  });
}

export function renderGroupMemberPicker() {
  const target = $("#groupMemberTools");
  if (!target) return;
  const members = groupMembers();
  const memberIds = new Set(members.map((member) => member.id || member.userId));
  target.classList.toggle("hidden", !canEditGroupMembers() || !state.groupMemberPickerOpen);
  target.innerHTML = state.groupMemberPickerOpen
    ? `
      <div class="group-member-picker-header" style="display: flex; gap: 8px; align-items: center; margin-bottom: 24px; margin-top: 4px; margin-left: -4px;">
        <button type="button" id="closeGroupMemberPicker" style="background: none; border: none; padding: 4px; margin: 0; cursor: pointer; color: #86868b; display: flex; align-items: center;" aria-label="Go back">
          <svg aria-hidden="true" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"></polyline></svg>
        </button>
        <h2 style="margin:0; font-size: 20px; font-weight: 600;">Add Members</h2>
      </div>
      ${renderMemberPicker({
        users: state.users,
        selectedIds: state.groupMemberSelection,
        search: state.groupMemberSearch,
        disabledIds: memberIds,
        searchId: "groupMemberSearch",
        optionAttribute: "data-group-member-option",
        removeAttribute: "data-remove-selected-group-member",
        submitId: "addGroupMember",
        emptyLabel: "No users found.",
      })}
    `
    : "";
}

export function renderGroupModal() {
  const modal = $("#groupModal");
  if (!modal) return;
  modal.classList.toggle("hidden", !state.groupModalOpen);
  if (!state.groupModalOpen) return;

  const mode = roleMode();
  const createAllowed = canCreateGroup();
  const editAllowed = canEditGroupMembers();
  const activeGroup = state.activeGroup || state.groups.find((group) => group.id === state.activeGroupId);
  const members = groupMembers();
  const modeCopy = {
    system: {
      title: "Group",
      listTitle: "All Groups",
      empty: "No groups yet. Create the first group to start assigning members.",
    },
    user: {
      title: "Group",
      listTitle: "My Groups",
      empty: "No groups available yet.",
    },
  }[mode];

  const creatingGroup = createAllowed && state.groupCreateOpen;
  modal.querySelector(".group-layout")?.classList.toggle("group-create-layout", creatingGroup);
  modal.querySelector(".group-detail")?.classList.toggle("group-create-detail", creatingGroup);
  modal.querySelector(".group-sidebar")?.classList.toggle("hidden", creatingGroup);
  $("#groupModalTitle").textContent = creatingGroup ? "New Group" : modeCopy.title;
  $("#groupListTitle").textContent = modeCopy.listTitle;
  $("#groupModalMeta").innerHTML = `
    <span>${creatingGroup ? "Add a name and initial members" : mode === "system" ? "Manage all groups" : "Groups you belong to"}</span>
  `;
  $("#groupCount").textContent = String(state.groups.length);
  $("#openGroupCreate").classList.toggle("hidden", !createAllowed || creatingGroup);
  $("#groupForm").classList.toggle("hidden", !creatingGroup);
  if (createAllowed) {
    renderGroupCreateMemberPicker();
  }

  const error = $("#groupError");
  error.classList.toggle("hidden", !state.groupError);
  error.textContent = state.groupError || "";

  if (creatingGroup) {
    $("#groupList").innerHTML = "";
    $("#groupDetailHeader").innerHTML = "";
    $("#groupProjectScope").innerHTML = "";
    $("#groupMembers").innerHTML = "";
    $("#groupMemberTools").innerHTML = "";
    $("#groupMemberTools").classList.add("hidden");
    return;
  }

  if (state.groupLoading) {
    $("#groupList").innerHTML = '<p class="project-meta">Loading groups...</p>';
    $("#groupDetailHeader").innerHTML = '<p class="project-meta">Loading group detail...</p>';
    $("#groupProjectScope").innerHTML = "";
    $("#groupMembers").innerHTML = "";
    $("#groupMemberTools").innerHTML = "";
    $("#groupMemberTools").classList.add("hidden");
    return;
  }

  $("#groupList").innerHTML = state.groups.length
    ? state.groups
        .map(
          (group) => `
        <button class="group-list-item ${group.id === state.activeGroupId ? "active" : ""}" type="button" data-select-group="${escapeHtml(group.id)}">
          <strong>${escapeHtml(group.name || "Untitled group")}</strong>
          <span>${group.memberCount ?? group.members?.length ?? 0}</span>
        </button>
      `,
        )
        .join("")
    : `<p class="project-meta">${modeCopy.empty}</p>`;

  if (!activeGroup) {
    $("#groupDetailHeader").innerHTML = "";
    $("#groupProjectScope").innerHTML = "";
    $("#groupMembers").innerHTML = "";
    $("#groupMemberTools").innerHTML = "";
    $("#groupMemberTools").classList.add("hidden");
    return;
  }

  $("#groupDetailHeader").innerHTML = `
    <div>
      <h3>${escapeHtml(activeGroup.name || "Untitled group")}</h3>
    </div>
    <div class="group-tabs" role="tablist">
      ${groupTabButton("members", "Members")}
      ${groupTabButton("projects", "Projects")}
    </div>
  `;

  const projects = groupProjects(activeGroup);
  const activeTab = state.groupDetailTab || "members";
  const showMemberTools = activeTab === "members" && state.groupMemberPickerOpen;
  $("#groupMemberTools").classList.toggle("hidden", !showMemberTools || !editAllowed);
  if (showMemberTools) {
    renderGroupMemberPicker();
  } else {
    $("#groupMemberTools").innerHTML = "";
  }

  if (activeTab === "members") {
    if (state.groupMemberPickerOpen) {
      $("#groupProjectScope").innerHTML = "";
      $("#groupMembers").innerHTML = "";
    } else {
      $("#groupProjectScope").innerHTML = `
        <div class="group-pane-actions">
          <button id="openGroupMemberPicker" class="add-member-button hidden" type="button" aria-label="Add member">
            <span aria-hidden="true">+</span>
            Add Member
          </button>
        </div>
      `;
      $("#openGroupMemberPicker").classList.toggle("hidden", !editAllowed);
      $("#groupMembers").innerHTML = renderMemberRows(members, editAllowed);
    }
    return;
  }

  $("#groupMemberTools").innerHTML = "";
  $("#groupMemberTools").classList.add("hidden");
  $("#groupProjectScope").innerHTML = "";
  $("#groupMembers").innerHTML = renderPlainList(projects, mode === "system" ? "No projects assigned." : "No project scope returned yet.");

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
  renderSharedConfigCreateModal();
  renderSharedConfigEditModal();
  renderConfigRows();
  renderConfigCreateDrawer();
  renderReviewDock();
  renderExportModal();
  renderReviewModal();
  renderRequests();
  renderVersionHistory();
  renderImportPreview();
  renderGroupModal();
}
