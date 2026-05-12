const state = {
  activeView: 'dashboard',
  activeProjectId: 'customer-portal',
  activeEnvironment: 'prod',
  configSearch: '',
  diffFilter: 'all',
  revealedKeys: new Set()
};

const navItems = [
  { id: 'dashboard', label: 'Dashboard', code: 'D' },
  { id: 'projects', label: 'Projects', code: 'P' },
  { id: 'templates', label: 'Templates', code: 'T' },
  { id: 'config', label: 'Config', code: 'C' },
  { id: 'diff', label: 'Diff', code: 'F' },
  { id: 'requests', label: 'Requests', code: 'R' }
];

const projects = [
  {
    id: 'customer-portal',
    name: 'customer-portal',
    owner: 'Platform Team',
    repoUrl: 'git.example.com/platform/customer-portal',
    defaultFormat: 'yaml',
    environments: ['dev', 'staging', 'prod'],
    health: 'Healthy',
    lastChanged: '12 min ago',
    configCount: 42
  },
  {
    id: 'billing-service',
    name: 'billing-service',
    owner: 'Finance Squad',
    repoUrl: 'git.example.com/payment/billing-service',
    defaultFormat: 'json',
    environments: ['dev', 'staging', 'prod'],
    health: 'Review',
    lastChanged: '48 min ago',
    configCount: 37
  },
  {
    id: 'retail-menu',
    name: 'retail-menu',
    owner: 'Retail Apps',
    repoUrl: 'git.example.com/store/retail-menu',
    defaultFormat: 'properties',
    environments: ['dev', 'qa', 'staging', 'prod'],
    health: 'Warning',
    lastChanged: '2 hr ago',
    configCount: 64
  }
];

const configs = [
  {
    projectId: 'customer-portal',
    environment: 'prod',
    key: 'api.baseUrl',
    value: 'https://api.example.com',
    type: 'string',
    isSensitive: false,
    status: 'Synced',
    updated: 'Alice Lin'
  },
  {
    projectId: 'customer-portal',
    environment: 'prod',
    key: 'database.url',
    value: 'postgresql://prod-user:secret@prod-db:5432/app',
    type: 'string',
    isSensitive: true,
    status: 'Protected',
    updated: 'Ben Wu'
  },
  {
    projectId: 'customer-portal',
    environment: 'prod',
    key: 'feature.newCheckoutEnabled',
    value: 'false',
    type: 'boolean',
    isSensitive: false,
    status: 'Changed',
    updated: 'Nora Chen'
  },
  {
    projectId: 'customer-portal',
    environment: 'prod',
    key: 'log.level',
    value: 'info',
    type: 'string',
    isSensitive: false,
    status: 'Synced',
    updated: 'Alice Lin'
  },
  {
    projectId: 'customer-portal',
    environment: 'staging',
    key: 'api.baseUrl',
    value: 'https://staging-api.example.com',
    type: 'string',
    isSensitive: false,
    status: 'Synced',
    updated: 'Alice Lin'
  },
  {
    projectId: 'customer-portal',
    environment: 'staging',
    key: 'database.url',
    value: 'postgresql://staging-user:secret@staging-db:5432/app',
    type: 'string',
    isSensitive: true,
    status: 'Protected',
    updated: 'Ben Wu'
  },
  {
    projectId: 'customer-portal',
    environment: 'staging',
    key: 'feature.newCheckoutEnabled',
    value: 'true',
    type: 'boolean',
    isSensitive: false,
    status: 'Changed',
    updated: 'Nora Chen'
  },
  {
    projectId: 'customer-portal',
    environment: 'dev',
    key: 'api.baseUrl',
    value: 'https://dev-api.example.com',
    type: 'string',
    isSensitive: false,
    status: 'Synced',
    updated: 'Alice Lin'
  },
  {
    projectId: 'billing-service',
    environment: 'prod',
    key: 'payment.gatewayUrl',
    value: 'https://gateway.example.com',
    type: 'string',
    isSensitive: false,
    status: 'Synced',
    updated: 'Ivy Kuo'
  },
  {
    projectId: 'billing-service',
    environment: 'prod',
    key: 'gateway.token',
    value: 'sk-live-prod-token',
    type: 'string',
    isSensitive: true,
    status: 'Protected',
    updated: 'Ivy Kuo'
  },
  {
    projectId: 'retail-menu',
    environment: 'prod',
    key: 'menu.cacheTtlSeconds',
    value: '300',
    type: 'number',
    isSensitive: false,
    status: 'Warning',
    updated: 'Ray Hsu'
  }
];

const templates = [
  {
    name: 'Base Application',
    format: 'yaml',
    keys: ['app.timezone', 'log.level', 'api.baseUrl', 'database.url'],
    description: 'Default service settings for standard web applications.'
  },
  {
    name: 'Feature Rollout',
    format: 'json',
    keys: ['feature.enabled', 'feature.rolloutPercent', 'feature.owner'],
    description: 'Feature flag and staged rollout settings.'
  },
  {
    name: 'Menu Service',
    format: 'properties',
    keys: ['menu.locale', 'menu.cacheTtlSeconds', 'menu.sourceUrl'],
    description: 'Shared menu and retail content configuration.'
  }
];

let requests = [
  {
    id: 'CR-1042',
    projectName: 'customer-portal',
    environment: 'prod',
    requester: 'Nora Chen',
    status: 'Pending',
    reason: 'Enable checkout flag after staging validation.'
  },
  {
    id: 'CR-1041',
    projectName: 'billing-service',
    environment: 'prod',
    requester: 'Ivy Kuo',
    status: 'Pending',
    reason: 'Rotate gateway token before release window.'
  },
  {
    id: 'CR-1038',
    projectName: 'retail-menu',
    environment: 'prod',
    requester: 'Ray Hsu',
    status: 'Approved',
    reason: 'Align menu cache TTL with global template.'
  }
];

const diffItems = [
  {
    key: 'api.baseUrl',
    status: 'modified',
    source: 'https://staging-api.example.com',
    target: 'https://api.example.com'
  },
  {
    key: 'feature.newCheckoutEnabled',
    status: 'modified',
    source: 'true',
    target: 'false'
  },
  {
    key: 'database.url',
    status: 'protected',
    source: '******',
    target: '******'
  },
  {
    key: 'log.format',
    status: 'added',
    source: 'json',
    target: 'missing'
  }
];

function $(selector) {
  return document.querySelector(selector);
}

function $all(selector) {
  return [...document.querySelectorAll(selector)];
}

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;');
}

function statusClass(status) {
  const normalized = status.toLowerCase();
  if (['healthy', 'synced', 'approved', 'valid'].includes(normalized)) {
    return 'success';
  }
  if (['review', 'changed', 'pending', 'warning', 'modified'].includes(normalized)) {
    return 'warning';
  }
  if (['blocked', 'failed', 'danger'].includes(normalized)) {
    return 'danger';
  }
  return 'neutral';
}

function activeProject() {
  return projects.find((project) => project.id === state.activeProjectId) ?? projects[0];
}

function switchView(viewId) {
  state.activeView = viewId;
  $all('.view').forEach((view) => {
    view.classList.toggle('active', view.dataset.view === viewId);
  });
  renderNav();
}

function showToast(message) {
  const toast = $('#toast');
  toast.textContent = message;
  toast.classList.add('show');
  window.setTimeout(() => toast.classList.remove('show'), 1800);
}

function renderNav() {
  $('#navList').innerHTML = navItems
    .map(
      (item) => `
        <button class="nav-item ${state.activeView === item.id ? 'active' : ''}" type="button" data-view-target="${item.id}">
          <span class="nav-code">${item.code}</span>
          <span class="nav-label">${item.label}</span>
        </button>
      `
    )
    .join('');
}

function renderStats() {
  const pendingCount = requests.filter((request) => request.status === 'Pending').length;
  const sensitiveCount = configs.filter((config) => config.isSensitive).length;
  const stats = [
    { label: 'Projects', value: projects.length, change: '+1 this week' },
    { label: 'Config Keys', value: configs.length, change: 'Across 4 environments' },
    { label: 'Pending Reviews', value: pendingCount, change: 'Prod guarded' },
    { label: 'Sensitive Keys', value: sensitiveCount, change: 'Masked by default' }
  ];

  $('#statsGrid').innerHTML = stats
    .map(
      (stat) => `
        <article class="stat-card">
          <p>${stat.label}</p>
          <strong>${stat.value}</strong>
          <span class="metric-change">${stat.change}</span>
        </article>
      `
    )
    .join('');
}

function renderDashboard() {
  renderStats();
  $('#dashboardProjects').innerHTML = projects
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
      `
    )
    .join('');

  $('#dashboardRequests').innerHTML = requests
    .filter((request) => request.status === 'Pending')
    .map(
      (request) => `
        <article class="project-row">
          <div>
            <h3>${escapeHtml(request.id)}</h3>
            <p class="project-meta">
              <span>${escapeHtml(request.projectName)}</span>
              <span>${escapeHtml(request.requester)}</span>
            </p>
          </div>
          <span class="status-pill warning">${request.status}</span>
        </article>
      `
    )
    .join('');
}

function renderProjects() {
  $('#projectsGrid').innerHTML = projects
    .map(
      (project) => `
        <article class="project-card">
          <div class="card-top">
            <div class="card-title">
              <h3>${escapeHtml(project.name)}</h3>
              <p>${escapeHtml(project.repoUrl)}</p>
            </div>
            <span class="status-pill ${statusClass(project.health)}">${project.health}</span>
          </div>
          <div>
            <p class="project-meta">
              <span>${escapeHtml(project.owner)}</span>
              <span>${project.configCount} config keys</span>
              <span>${escapeHtml(project.defaultFormat)}</span>
            </p>
          </div>
          <div class="environment-strip">
            ${project.environments.map((environment) => `<span>${environment}</span>`).join('')}
          </div>
          <div class="card-actions">
            <button class="secondary-action" type="button" data-open-config="${project.id}">Open Config</button>
            <button class="ghost-action" type="button" data-open-diff="${project.id}">Compare</button>
          </div>
        </article>
      `
    )
    .join('');
}

function renderTemplates() {
  $('#templatesGrid').innerHTML = templates
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
                `
              )
              .join('')}
          </div>
          <button class="secondary-action" type="button" data-template="${escapeHtml(template.name)}">Apply</button>
        </article>
      `
    )
    .join('');
}

function renderConfigProjectList() {
  $('#configProjectList').innerHTML = projects
    .map(
      (project) => `
        <button class="compact-item ${project.id === state.activeProjectId ? 'active' : ''}" type="button" data-select-project="${project.id}">
          <div>
            <h3>${escapeHtml(project.name)}</h3>
            <p class="project-meta">
              <span>${escapeHtml(project.owner)}</span>
              <span>${project.configCount} keys</span>
            </p>
          </div>
          <span class="format-pill">${escapeHtml(project.defaultFormat)}</span>
        </button>
      `
    )
    .join('');
}

function renderEnvironmentTabs() {
  const project = activeProject();
  if (!project.environments.includes(state.activeEnvironment)) {
    state.activeEnvironment = project.environments[0];
  }

  $('#environmentTabs').innerHTML = project.environments
    .map(
      (environment) => `
        <button class="${environment === state.activeEnvironment ? 'active' : ''}" type="button" data-env="${environment}">
          ${environment}
        </button>
      `
    )
    .join('');
}

function renderConfigRows() {
  const project = activeProject();
  $('#configTitle').textContent = project.name;
  renderConfigProjectList();
  renderEnvironmentTabs();

  const search = state.configSearch.trim().toLowerCase();
  const rows = configs
    .filter(
      (config) =>
        config.projectId === state.activeProjectId &&
        config.environment === state.activeEnvironment
    )
    .filter((config) => {
      if (!search) return true;
      return (
        config.key.toLowerCase().includes(search) ||
        config.value.toLowerCase().includes(search)
      );
    });

  $('#configRows').innerHTML =
    rows
      .map((config) => {
        const revealKey = `${config.projectId}:${config.environment}:${config.key}`;
        const visibleValue =
          config.isSensitive && !state.revealedKeys.has(revealKey)
            ? '******'
            : config.value;
        return `
          <tr>
            <td class="key-cell">${escapeHtml(config.key)}</td>
            <td class="value-cell">${escapeHtml(visibleValue)}</td>
            <td>${escapeHtml(config.type)}</td>
            <td><span class="status-pill ${statusClass(config.status)}">${escapeHtml(config.status)}</span></td>
            <td>
              <span>${escapeHtml(config.updated)}</span>
              ${
                config.isSensitive
                  ? `<div class="row-actions"><button class="tiny-button" type="button" data-reveal="${escapeHtml(revealKey)}">${state.revealedKeys.has(revealKey) ? 'Hide' : 'Reveal'}</button></div>`
                  : ''
              }
            </td>
          </tr>
        `;
      })
      .join('') ||
    `<tr><td colspan="5" class="value-cell">No config keys match this view.</td></tr>`;
}

function renderDiffFilters() {
  const filters = ['all', 'modified', 'added', 'protected'];
  $('#diffFilters').innerHTML = filters
    .map(
      (filter) => `
        <button class="${filter === state.diffFilter ? 'active' : ''}" type="button" data-diff-filter="${filter}">
          ${filter}
        </button>
      `
    )
    .join('');
}

function renderDiff() {
  renderDiffFilters();
  const items =
    state.diffFilter === 'all'
      ? diffItems
      : diffItems.filter((item) => item.status === state.diffFilter);

  $('#validationBadge').textContent = 'Valid';
  $('#diffList').innerHTML = items
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
      `
    )
    .join('');
}

function renderRequests() {
  $('#notificationCount').textContent = String(
    requests.filter((request) => request.status === 'Pending').length
  );
  $('#requestList').innerHTML = requests
    .map(
      (request) => `
        <article class="request-item">
          <div>
            <div class="project-meta">
              <span>${escapeHtml(request.id)}</span>
              <span>${escapeHtml(request.projectName)}</span>
              <span>${escapeHtml(request.environment)}</span>
            </div>
            <h3>${escapeHtml(request.reason)}</h3>
            <p>${escapeHtml(request.requester)}</p>
          </div>
          <div class="request-actions">
            <span class="status-pill ${statusClass(request.status)}">${escapeHtml(request.status)}</span>
            ${
              request.status === 'Pending'
                ? `<button class="secondary-action" type="button" data-approve="${escapeHtml(request.id)}">Approve</button>
                   <button class="ghost-action" type="button" data-reject="${escapeHtml(request.id)}">Reject</button>`
                : ''
            }
          </div>
        </article>
      `
    )
    .join('');
}

function renderAll() {
  renderNav();
  renderDashboard();
  renderProjects();
  renderTemplates();
  renderConfigRows();
  renderDiff();
  renderRequests();
}

document.addEventListener('click', (event) => {
  const target = event.target.closest('button');
  if (!target) return;

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
    switchView('config');
    renderConfigRows();
    return;
  }

  if (target.dataset.openDiff) {
    state.activeProjectId = target.dataset.openDiff;
    switchView('diff');
    return;
  }

  if (target.dataset.env) {
    state.activeEnvironment = target.dataset.env;
    renderConfigRows();
    return;
  }

  if (target.dataset.reveal) {
    const key = target.dataset.reveal;
    if (state.revealedKeys.has(key)) {
      state.revealedKeys.delete(key);
    } else {
      state.revealedKeys.add(key);
    }
    renderConfigRows();
    return;
  }

  if (target.dataset.diffFilter) {
    state.diffFilter = target.dataset.diffFilter;
    renderDiff();
    return;
  }

  if (target.dataset.approve || target.dataset.reject) {
    const id = target.dataset.approve ?? target.dataset.reject;
    requests = requests.map((request) =>
      request.id === id
        ? { ...request, status: target.dataset.approve ? 'Approved' : 'Rejected' }
        : request
    );
    renderDashboard();
    renderRequests();
    showToast(`${id} ${target.dataset.approve ? 'approved' : 'rejected'}`);
    return;
  }

  if (target.dataset.template) {
    showToast(`${target.dataset.template} template queued`);
    return;
  }
});

$('#configSearch').addEventListener('input', (event) => {
  state.configSearch = event.target.value;
  renderConfigRows();
});

$('#globalSearch').addEventListener('input', (event) => {
  const value = event.target.value.trim().toLowerCase();
  if (!value) return;
  const matchedProject = projects.find((project) =>
    [project.name, project.owner, project.repoUrl].some((field) =>
      field.toLowerCase().includes(value)
    )
  );
  if (matchedProject) {
    state.activeProjectId = matchedProject.id;
    renderConfigRows();
  }
});

$('#mockRegisterProject').addEventListener('click', () => {
  showToast('Project registration draft created');
});

$('#submitReview').addEventListener('click', () => {
  const project = activeProject();
  const requestId = `CR-${1043 + requests.length}`;
  requests = [
    {
      id: requestId,
      projectName: project.name,
      environment: state.activeEnvironment,
      requester: 'Alice Lin',
      status: 'Pending',
      reason: `Review ${state.activeEnvironment} changes for ${project.name}.`
    },
    ...requests
  ];
  renderDashboard();
  renderRequests();
  showToast(`${requestId} submitted`);
});

$('#exportReport').addEventListener('click', () => {
  showToast('Validation report exported');
});

renderAll();
