import { $ } from './dom.js';
import { activeProject, navItems, state } from './state.js';
import {
  escapeHtml,
  formatDateTime,
  initials,
  statusClass
} from './utils.js';

export function renderNav() {
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

export function renderUser() {
  $('#userInitials').textContent = initials(state.user?.name || '--');
  $('#userName').textContent = `${state.user?.name || 'Not signed in'} · ${state.user?.role || ''}`;
}

export function renderStats() {
  const pendingCount = state.requests.filter(
    (request) => request.status === 'pending'
  ).length;
  const sensitiveCount = state.configs.filter((config) => config.isSensitive).length;
  const stats = [
    { label: 'Projects', value: state.projects.length, change: 'Live from API' },
    {
      label: 'Active Keys',
      value: state.configs.length,
      change: state.activeEnvironment
    },
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

export function renderDashboard() {
  renderStats();
  $('#dashboardProjects').innerHTML = state.projects
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
        </article>
      `
    )
    .join('');

  $('#dashboardRequests').innerHTML =
    state.requests
      .filter((request) => request.status === 'pending')
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
        `
      )
      .join('') || '<p class="project-meta">No pending review requests.</p>';
}

export function renderProjects() {
  $('#projectsGrid').innerHTML =
    state.projects
      .map(
        (project) => `
          <button class="project-card project-card-button" type="button" data-open-config="${project.id}">
            <div class="card-top">
              <div class="card-title">
                <h3>${escapeHtml(project.name)}</h3>
                <p>${escapeHtml(project.repoUrl || 'No repository URL')}</p>
              </div>
            </div>
            <p class="project-meta">
              <span>${escapeHtml(project.owner)}</span>
              <span>${project.configCount} config keys</span>
              <span>${escapeHtml(project.defaultFormat)}</span>
            </p>
            <div class="environment-strip">
              ${project.environments.map((environment) => `<span>${environment}</span>`).join('')}
            </div>
          </button>
        `
      )
      .join('') || '<p>No projects yet. Create one from the backend API.</p>';
}

export function renderTemplates() {
  $('#templatesGrid').innerHTML = state.templates
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
        </article>
      `
    )
    .join('');
}

export function renderConfigProjectList() {
  $('#configProjectList').innerHTML = state.projects
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

export function renderEnvironmentTabs() {
  const project = activeProject();
  if (!project) {
    $('#environmentTabs').innerHTML = '';
    return;
  }
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

export function renderConfigRows() {
  const project = activeProject();
  $('#configTitle').textContent = project?.name || 'Project Config';
  renderConfigVersionLabel();
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
            <td>${escapeHtml(config.updated)}</td>
            <td>
              <div class="row-actions">
                ${
                  config.isSensitive
                    ? `<button class="tiny-button" type="button" data-reveal="${escapeHtml(revealKey)}">${state.revealedKeys.has(revealKey) ? 'Hide' : 'Reveal'}</button>`
                    : ''
                }
                <button class="tiny-button" type="button" data-edit-config="${escapeHtml(config.id)}">Edit</button>
              </div>
            </td>
          </tr>
        `;
      })
      .join('') ||
    `<tr><td colspan="4" class="value-cell">No config keys match this view.</td></tr>`;
}


export function renderRequests() {
  $('#notificationCount').textContent = String(
    state.requests.filter((request) => request.status === 'pending').length
  );
  $('#requestList').innerHTML =
    state.requests
      .map(
        (request) => `
          <article class="request-item">
            <div>
              <div class="project-meta">
                <span>${escapeHtml(request.id.slice(0, 8))}</span>
                <span>${escapeHtml(request.projectName)}</span>
                <span>${escapeHtml(request.environment)}</span>
                ${request.configKey ? `<span>${escapeHtml(request.configKey)}</span>` : ''}
              </div>
              <h3>${escapeHtml(request.reason)}</h3>
              <p>${escapeHtml(request.requester)}</p>
            </div>
            <div class="request-actions">
              <span class="status-pill ${statusClass(request.status)}">${escapeHtml(request.status)}</span>
              ${
                request.status === 'pending' && ['system_admin', 'reviewer'].includes(state.user?.role)
                  ? `<button class="secondary-action" type="button" data-approve="${escapeHtml(request.id)}">Approve</button>
                     <button class="ghost-action" type="button" data-reject="${escapeHtml(request.id)}">Reject</button>`
                  : ''
              }
            </div>
          </article>
        `
      )
      .join('') || '<p class="project-meta">No review requests yet.</p>';
}

export function renderVersionHistory() {
  $('#historyModal').classList.toggle('hidden', !state.historyModalOpen);
  if (!state.historyModalOpen) return;

  const project = activeProject();
  const current = state.configHistory[0];
  const previous = state.configHistory[1];
  $('#historyTitle').textContent = `${project?.name || 'Config'} History`;
  $('#historyMeta').innerHTML = project
    ? `<span>${escapeHtml(project.name)}</span><span>${escapeHtml(state.activeEnvironment)}</span>`
    : '';

  if (state.historyLoading) {
    $('#historySummary').innerHTML = '<p class="project-meta">Loading config history...</p>';
    $('#versionList').innerHTML = '';
    $('#rollbackLatest').disabled = true;
    return;
  }

  $('#rollbackLatest').disabled = !previous;
  $('#historySummary').innerHTML = current
    ? `
      <div class="history-previous current">
        <span>Current Config</span>
        <code>${current.entries.length} keys · ${escapeHtml(current.changeReason)}</code>
      </div>
      <div class="history-previous">
        <span>Previous Config</span>
        <code>${previous ? `${previous.entries.length} keys · ${escapeHtml(previous.changeReason)}` : 'No previous snapshot'}</code>
      </div>
    `
    : '<p class="project-meta">No config history yet.</p>';

  $('#versionList').innerHTML = state.configHistory
    .map(
      (snapshot, index) => {
        const version = snapshot.id ? snapshot.id.slice(0, 12) : 'current';
        return `
          <article class="version-item version-record">
            <div class="version-record-line">
              <strong>${escapeHtml(snapshot.changedBy)}</strong>
              <span>${index === 0 ? 'current version' : 'version'}</span>
              <code>${escapeHtml(version)}</code>
              <span>${escapeHtml(formatDateTime(snapshot.createdAt))}</span>
            </div>
            <div class="version-record-meta">
              <span>${snapshot.entries.length} keys</span>
              <span>${escapeHtml(snapshot.changeReason)}</span>
            </div>
          </article>
        `;
      }
    )
    .join('');
}

function renderConfigVersionLabel() {
  const label = $('#configVersionLabel');
  if (!label) return;

  const project = activeProject();
  if (!project) {
    label.textContent = 'No project selected';
    return;
  }

  const current = state.configHistory[0];
  if (!current) {
    label.textContent = `${state.activeEnvironment} · No saved version yet`;
    return;
  }

  const version = current.id ? current.id.slice(0, 12) : 'current';
  label.textContent = `${state.activeEnvironment} · Version ${version} · ${formatDateTime(current.createdAt)} · ${current.entries.length} keys`;
}

export function renderImportPreview() {
  $('#importPreviewModal').classList.toggle('hidden', !state.importPreviewOpen);
  if (!state.importPreviewOpen) return;

  const preview = state.importPreview;
  $('#applyImportConfig').disabled = state.importApplying || !preview;
  if (!preview) {
    $('#importPreviewTitle').textContent = 'Extracted Config';
    $('#importPreviewMeta').textContent = '';
    $('#importPreviewSummary').innerHTML = '<p class="project-meta">No extracted config yet.</p>';
    $('#importPreviewList').innerHTML = '';
    return;
  }

  $('#importPreviewTitle').textContent = `Extracted ${preview.fileName}`;
  $('#importPreviewMeta').innerHTML = `
    <span>${escapeHtml(preview.environment)}</span>
    <span>${escapeHtml(preview.format)}</span>
    <span>${preview.entryCount} keys</span>
  `;
  $('#importPreviewSummary').innerHTML = `
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
  $('#importPreviewList').innerHTML = preview.entries
    .map(
      (entry) => `
        <article class="snapshot-entry import-preview-entry">
          <span>${escapeHtml(entry.key)}</span>
          <code>${escapeHtml(entry.value)}</code>
        </article>
      `
    )
    .join('');
}


export function renderAll() {
  renderUser();
  renderNav();
  renderDashboard();
  renderProjects();
  renderTemplates();
  renderConfigRows();
  renderRequests();
  renderVersionHistory();
  renderImportPreview();
}
