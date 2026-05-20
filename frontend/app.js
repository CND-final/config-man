const API_BASE = "/api/v1";

const state = {
  token: localStorage.getItem("configManToken") || "",
  user: JSON.parse(localStorage.getItem("configManUser") || "null"),
  activeView: "dashboard",
  activeProjectId: "",
  activeEnvironment: "prod",
  configSearch: "",
  diffFilter: "all",
  revealedKeys: new Set(),
  projects: [],
  configs: [],
  requests: [],
  templates: [],
  diffItems: [],
};

const navItems = [
  { id: "dashboard", label: "Dashboard", code: "D" },
  { id: "projects", label: "Projects", code: "P" },
  { id: "templates", label: "Templates", code: "T" },
  { id: "config", label: "Config", code: "C" },
  { id: "diff", label: "Diff", code: "F" },
  { id: "requests", label: "Requests", code: "R" },
];

function $(selector) {
  return document.querySelector(selector);
}

function $all(selector) {
  return [...document.querySelectorAll(selector)];
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function statusClass(status) {
  const normalized = String(status).toLowerCase();
  if (["healthy", "synced", "approved", "valid"].includes(normalized)) {
    return "success";
  }
  if (
    ["review", "changed", "pending", "warning", "modified"].includes(normalized)
  ) {
    return "warning";
  }
  if (["blocked", "failed", "danger", "rejected"].includes(normalized)) {
    return "danger";
  }
  return "neutral";
}

async function api(path, options = {}) {
  const url = `${API_BASE}${path}`;
  let response;
  try {
    response = await fetch(url, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...(state.token ? { Authorization: `Bearer ${state.token}` } : {}),
        ...(options.headers || {}),
      },
    });
  } catch (error) {
    console.error("Network error while calling API:", url, error);
    throw new Error(
      `Cannot reach API ${url}. Check backend is running and configManApiBase is correct.`,
    );
  }

  const text = await response.text();
  const data = text ? JSON.parse(text) : null;
  if (!response.ok) {
    throw new Error(data?.message || `API request failed: ${response.status}`);
  }
  return data;
}

function activeProject() {
  return (
    state.projects.find((project) => project.id === state.activeProjectId) ||
    state.projects[0]
  );
}

function showToast(message) {
  const toast = $("#toast");
  toast.textContent = message;
  toast.classList.add("show");
  window.setTimeout(() => toast.classList.remove("show"), 2200);
}

function setAuthenticated(authenticated) {
  $("#loginScreen").classList.toggle("hidden", authenticated);
  $("#appShell").classList.toggle("hidden", !authenticated);
}

function switchView(viewId) {
  state.activeView = viewId;
  $all(".view").forEach((view) => {
    view.classList.toggle("active", view.dataset.view === viewId);
  });
  renderNav();
  if (viewId === "diff") {
    loadDiff().catch((error) => showToast(error.message));
  }
}

function initials(name) {
  return String(name)
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();
}

function normalizeProject(project) {
  return {
    ...project,
    owner: project.ownerName,
    environments: project.environments.map((environment) =>
      typeof environment === "string" ? environment : environment.name,
    ),
    health: project.configCount > 0 ? "Healthy" : "Review",
    lastChanged: "live API",
    configCount: project.configCount ?? 0,
  };
}

async function loadInitialData() {
  const [projects, template, requests] = await Promise.all([
    api("/projects"),
    api("/templates/base"),
    api("/review-requests"),
  ]);

  state.projects = projects.map(normalizeProject);
  state.templates = [
    {
      name: template.name,
      format: "base",
      description: "Required baseline keys from the backend template.",
      keys: template.entries.map((entry) => entry.key),
    },
  ];
  state.requests = requests;

  if (!state.activeProjectId && state.projects[0]) {
    state.activeProjectId = state.projects[0].id;
  }

  const project = activeProject();
  if (project && !project.environments.includes(state.activeEnvironment)) {
    state.activeEnvironment = project.environments[0];
  }

  await loadConfigs();
  await loadDiff();
  renderAll();
}

async function loadConfigs(revealSensitive = false) {
  const project = activeProject();
  if (!project) {
    state.configs = [];
    return;
  }

  const data = await api(
    `/projects/${project.id}/configs?env=${encodeURIComponent(
      state.activeEnvironment,
    )}${revealSensitive ? "&revealSensitive=true" : ""}`,
  );
  state.configs = data.entries.map((entry) => ({
    ...entry,
    type: entry.valueType,
    status: entry.isSensitive ? "Protected" : "Synced",
    updated: entry.updatedBy,
  }));
}

async function loadDiff() {
  const project = activeProject();
  if (
    !project ||
    !project.environments.includes("staging") ||
    !project.environments.includes("prod")
  ) {
    state.diffItems = [];
    renderDiff();
    return;
  }

  const [staging, prod] = await Promise.all([
    api(`/projects/${project.id}/configs?env=staging`),
    api(`/projects/${project.id}/configs?env=prod`),
  ]);
  const stagingMap = new Map(
    staging.entries.map((entry) => [entry.key, entry]),
  );
  const prodMap = new Map(prod.entries.map((entry) => [entry.key, entry]));
  const keys = [...new Set([...stagingMap.keys(), ...prodMap.keys()])].sort();

  state.diffItems = keys
    .map((key) => {
      const source = stagingMap.get(key);
      const target = prodMap.get(key);
      const status = !target
        ? "added"
        : !source
          ? "removed"
          : source.value !== target.value
            ? "modified"
            : "synced";
      return {
        key,
        status:
          source?.isSensitive || target?.isSensitive ? "protected" : status,
        source: source?.value ?? "missing",
        target: target?.value ?? "missing",
      };
    })
    .filter((item) => item.status !== "synced");

  renderDiff();
}

function renderNav() {
  $("#navList").innerHTML = navItems
    .map(
      (item) => `
        <button class="nav-item ${state.activeView === item.id ? "active" : ""}" type="button" data-view-target="${item.id}">
          <span class="nav-code">${item.code}</span>
          <span class="nav-label">${item.label}</span>
        </button>
      `,
    )
    .join("");
}

function renderUser() {
  $("#userInitials").textContent = initials(state.user?.name || "--");
  $("#userName").textContent =
    `${state.user?.name || "Not signed in"} · ${state.user?.role || ""}`;
}

function renderStats() {
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

function renderDashboard() {
  renderStats();
  $("#dashboardProjects").innerHTML = state.projects
    .slice(0, 3)
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
          <span class="status-pill ${statusClass(project.health)}">${project.health}</span>
        </article>
      `,
    )
    .join("");

  $("#dashboardRequests").innerHTML =
    state.requests
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

function renderProjects() {
  $("#projectsGrid").innerHTML =
    state.projects
      .map(
        (project) => `
          <article class="project-card">
            <div class="card-top">
              <div class="card-title">
                <h3>${escapeHtml(project.name)}</h3>
                <p>${escapeHtml(project.repoUrl || "No repository URL")}</p>
              </div>
              <span class="status-pill ${statusClass(project.health)}">${project.health}</span>
            </div>
            <p class="project-meta">
              <span>${escapeHtml(project.owner)}</span>
              <span>${project.configCount} config keys</span>
              <span>${escapeHtml(project.defaultFormat)}</span>
            </p>
            <div class="environment-strip">
              ${project.environments.map((environment) => `<span>${environment}</span>`).join("")}
            </div>
            <div class="card-actions">
              <button class="secondary-action" type="button" data-open-config="${project.id}">Open Config</button>
              <button class="ghost-action" type="button" data-open-diff="${project.id}">Compare</button>
            </div>
          </article>
        `,
      )
      .join("") || "<p>No projects yet. Create one from the backend API.</p>";
}

function renderTemplates() {
  $("#templatesGrid").innerHTML = state.templates
    .map(
      (template) => `
        <article class="template-card">
          <div class="card-top">
            <div class="card-title">
              <h3>${escapeHtml(template.name)}</h3>
              <p>${escapeHtml(template.description)}</p>
            </div>
            <span class="format-pill">${escapeHtml(template.format)}</span>
          </div>
          <div class="template-list">
            ${template.keys
              .map(
                (key) => `
                  <div class="template-key">
                    <span>${escapeHtml(key)}</span>
                    <span class="status-pill neutral">base</span>
                  </div>
                `,
              )
              .join("")}
          </div>
        </article>
      `,
    )
    .join("");
}

function renderConfigProjectList() {
  $("#configProjectList").innerHTML = state.projects
    .map(
      (project) => `
        <button class="compact-item ${project.id === state.activeProjectId ? "active" : ""}" type="button" data-select-project="${project.id}">
          <div>
            <h3>${escapeHtml(project.name)}</h3>
            <p class="project-meta">
              <span>${escapeHtml(project.owner)}</span>
              <span>${project.configCount} keys</span>
            </p>
          </div>
          <span class="format-pill">${escapeHtml(project.defaultFormat)}</span>
        </button>
      `,
    )
    .join("");
}

function renderEnvironmentTabs() {
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

function renderConfigRows() {
  const project = activeProject();
  $("#configTitle").textContent = project?.name || "Project Config";
  renderConfigProjectList();
  renderEnvironmentTabs();

  const search = state.configSearch.trim().toLowerCase();
  const rows = state.configs.filter((config) => {
    if (!search) return true;
    return (
      config.key.toLowerCase().includes(search) ||
      String(config.value).toLowerCase().includes(search)
    );
  });

  $("#configRows").innerHTML =
    rows
      .map((config) => {
        const revealKey = `${config.projectId}:${config.environment}:${config.key}`;
        const visibleValue =
          config.isSensitive && !state.revealedKeys.has(revealKey)
            ? "******"
            : config.value;
        return `
          <tr>
            <td class="key-cell">${escapeHtml(config.key)}</td>
            <td class="value-cell">${escapeHtml(visibleValue)}</td>
            <td>${escapeHtml(config.type)}</td>
            <td><span class="status-pill ${statusClass(config.status)}">${escapeHtml(config.status)}</span></td>
            <td>${escapeHtml(config.updated)}</td>
            <td>
              <div class="row-actions">
                ${
                  config.isSensitive
                    ? `<button class="tiny-button" type="button" data-reveal="${escapeHtml(revealKey)}">${state.revealedKeys.has(revealKey) ? "Hide" : "Reveal"}</button>`
                    : ""
                }
                <button class="tiny-button" type="button" data-edit-config="${escapeHtml(config.id)}">Edit</button>
              </div>
            </td>
          </tr>
        `;
      })
      .join("") ||
    `<tr><td colspan="6" class="value-cell">No config keys match this view.</td></tr>`;
}

function renderDiffFilters() {
  const filters = ["all", "modified", "added", "removed", "protected"];
  $("#diffFilters").innerHTML = filters
    .map(
      (filter) => `
        <button class="${filter === state.diffFilter ? "active" : ""}" type="button" data-diff-filter="${filter}">
          ${filter}
        </button>
      `,
    )
    .join("");
}

function renderDiff() {
  renderDiffFilters();
  const items =
    state.diffFilter === "all"
      ? state.diffItems
      : state.diffItems.filter((item) => item.status === state.diffFilter);

  $("#validationBadge").textContent = state.diffItems.length
    ? "Review"
    : "Valid";
  $("#validationBadge").className =
    `status-pill ${state.diffItems.length ? "warning" : "success"}`;
  $("#diffList").innerHTML =
    items
      .map(
        (item) => `
          <article class="diff-item">
            <div class="diff-label">
              <span class="status-pill ${statusClass(item.status)}">${item.status}</span>
              <h3>${escapeHtml(item.key)}</h3>
            </div>
            <div class="diff-value">
              <strong>staging</strong>
              <code>${escapeHtml(item.source)}</code>
            </div>
            <div class="diff-value">
              <strong>prod</strong>
              <code>${escapeHtml(item.target)}</code>
            </div>
          </article>
        `,
      )
      .join("") || '<p class="project-meta">No environment differences.</p>';
}

function renderRequests() {
  $("#notificationCount").textContent = String(
    state.requests.filter((request) => request.status === "pending").length,
  );
  $("#requestList").innerHTML =
    state.requests
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

function renderAll() {
  renderUser();
  renderNav();
  renderDashboard();
  renderProjects();
  renderTemplates();
  renderConfigRows();
  renderDiff();
  renderRequests();
}

async function editConfig(configId) {
  const config = state.configs.find((entry) => entry.id === configId);
  if (!config) return;

  if (isProdSensitiveEdit(config)) {
    const hasReview = await hasReviewRequest(config);
    if (!hasReview) {
      const confirmed = window.confirm(
        "此為敏感環境，是否已建立一筆 Review Request？",
      );
      if (!confirmed) {
        return;
      }
    }
  }

  const nextValue = window.prompt(`Update ${config.key}`, config.value);
  if (nextValue === null || nextValue === config.value) {
    return;
  }

  await api(`/projects/${config.projectId}/configs/${config.id}`, {
    method: "PUT",
    body: JSON.stringify({
      value: nextValue,
      changeReason: "updated from frontend prototype",
    }),
  });

  await loadConfigs();
  await loadDiff();
  renderAll();
  showToast(`${config.key} updated`);
}

function isProdSensitiveEdit(config) {
  return (
    config.environment === "prod" &&
    (config.isSensitive ||
      /(database|db).*(password|url)|password|secret|token/i.test(config.key))
  );
}

async function hasReviewRequest(config) {
  const requests = await api(
    `/projects/${config.projectId}/review-requests?env=prod&key=${encodeURIComponent(
      config.key,
    )}&status=pending`,
  );
  return requests.length > 0;
}

async function importConfigFile() {
  const file = $("#configFile").files[0];
  const project = activeProject();
  if (!file || !project) {
    showToast("Choose a config file first");
    return;
  }

  const format = $("#configFormat").value;
  const content = await file.text();
  const result = await api(`/projects/${project.id}/configs/import`, {
    method: "POST",
    body: JSON.stringify({
      environment: state.activeEnvironment,
      format,
      content,
      changeReason: `import ${file.name}`,
    }),
  });

  await loadConfigs();
  await loadDiff();
  renderAll();
  showToast(
    `Imported ${result.imported}: ${result.created} created, ${result.updated} updated`,
  );
}

async function createReviewRequest() {
  const project = activeProject();
  if (!project) return;
  const configKey = window.prompt(
    "Config key for review request",
    "database.url",
  );
  if (configKey === null) return;
  const reason = window.prompt(
    "Reason",
    `Review ${state.activeEnvironment} change for ${project.name}`,
  );
  if (!reason) return;

  await api("/review-requests", {
    method: "POST",
    body: JSON.stringify({
      projectId: project.id,
      environment: state.activeEnvironment,
      configKey,
      reason,
    }),
  });

  state.requests = await api("/review-requests");
  renderDashboard();
  renderRequests();
  showToast("Review request created");
}

async function handleReviewDecision(id, action) {
  await api(`/review-requests/${id}/${action}`, {
    method: "PUT",
    body: JSON.stringify({ comment: `${action} from frontend` }),
  });
  state.requests = await api("/review-requests");
  renderDashboard();
  renderRequests();
  showToast(`Review request ${action}d`);
}

document.addEventListener("click", async (event) => {
  const target = event.target.closest("button");
  if (!target) return;

  try {
    const jump = target.dataset.jump;
    if (jump) {
      switchView(jump);
      return;
    }

    const viewTarget = target.dataset.viewTarget;
    if (viewTarget) {
      switchView(viewTarget);
      return;
    }

    const projectId = target.dataset.openConfig ?? target.dataset.selectProject;
    if (projectId) {
      state.activeProjectId = projectId;
      const project = activeProject();
      state.activeEnvironment = project.environments.includes("prod")
        ? "prod"
        : project.environments[0];
      await loadConfigs();
      switchView("config");
      renderAll();
      return;
    }

    if (target.dataset.openDiff) {
      state.activeProjectId = target.dataset.openDiff;
      switchView("diff");
      return;
    }

    if (target.dataset.env) {
      state.activeEnvironment = target.dataset.env;
      await loadConfigs();
      renderAll();
      return;
    }

    if (target.dataset.reveal) {
      const key = target.dataset.reveal;
      if (state.revealedKeys.has(key)) {
        state.revealedKeys.delete(key);
        await loadConfigs();
      } else {
        state.revealedKeys.add(key);
        await loadConfigs(true);
      }
      renderConfigRows();
      return;
    }

    if (target.dataset.editConfig) {
      await editConfig(target.dataset.editConfig);
      return;
    }

    if (target.dataset.diffFilter) {
      state.diffFilter = target.dataset.diffFilter;
      renderDiff();
      return;
    }

    if (target.dataset.approve) {
      await handleReviewDecision(target.dataset.approve, "approve");
      return;
    }

    if (target.dataset.reject) {
      await handleReviewDecision(target.dataset.reject, "reject");
    }
  } catch (error) {
    showToast(error.message);
  }
});

$("#loginForm").addEventListener("submit", async (event) => {
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
    showToast(`Signed in as ${state.user.name}`);
  } catch (error) {
    showToast(error.message);
  }
});

$("#logoutButton").addEventListener("click", () => {
  localStorage.removeItem("configManToken");
  localStorage.removeItem("configManUser");
  state.token = "";
  state.user = null;
  setAuthenticated(false);
});

$("#configSearch").addEventListener("input", (event) => {
  state.configSearch = event.target.value;
  renderConfigRows();
});

$("#configFile").addEventListener("change", (event) => {
  const name = event.target.files[0]?.name || "";
  const ext = name.split(".").pop()?.toLowerCase();
  if (ext === "json") $("#configFormat").value = "json";
  if (ext === "yaml" || ext === "yml") $("#configFormat").value = "yaml";
  if (ext === "properties") $("#configFormat").value = "properties";
});

$("#importConfig").addEventListener("click", () => {
  importConfigFile().catch((error) => showToast(error.message));
});

$("#submitReview").addEventListener("click", () => {
  createReviewRequest().catch((error) => showToast(error.message));
});

$("#exportReport").addEventListener("click", () => {
  showToast("Validation report exported");
});

$("#mockRegisterProject").addEventListener("click", () => {
  showToast("Project registration now uses POST /api/v1/projects");
});

if (state.token && state.user) {
  $("#apiBaseLabel").textContent = API_BASE;
  setAuthenticated(true);
  loadInitialData().catch((error) => {
    showToast(error.message);
    setAuthenticated(false);
  });
} else {
  $("#apiBaseLabel").textContent = API_BASE;
  setAuthenticated(false);
}
