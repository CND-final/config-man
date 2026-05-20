import { api } from './api.js';
import { $, $all, showToast } from './dom.js';
import {
  loadConfigHistory,
  loadConfigs,
  loadDiff,
  reloadProjects
} from './data.js';
import {
  renderAll,
  renderConfigRows,
  renderDashboard,
  renderDiff,
  renderNav,
  renderRequests,
  renderVersionHistory
} from './render.js';
import { activeProject, state } from './state.js';
import {
  isProdSensitiveEdit,
  parseEnvironmentInput
} from './utils.js';

export function switchView(viewId) {
  state.activeView = viewId;
  $all('.view').forEach((view) => {
    view.classList.toggle('active', view.dataset.view === viewId);
  });
  renderNav();
  if (viewId === 'diff') {
    loadDiff()
      .then(renderDiff)
      .catch((error) => showToast(error.message));
  }
}

export function setProjectModal(open) {
  state.projectModalOpen = open;
  $('#projectModal').classList.toggle('hidden', !open);
  if (open) {
    window.setTimeout(() => $('#projectName').focus(), 0);
  } else {
    $('#projectForm').reset();
    $('#projectEnvironments').value = 'dev, staging, prod';
  }
}

export function setHistoryModal(open) {
  state.historyModalOpen = open;
  if (!open) {
    state.configHistory = [];
    state.historyLoading = false;
  }
  renderVersionHistory();
}

export async function openVersionHistory() {
  state.configHistory = [];
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
  const created = await api('/projects', {
    method: 'POST',
    body: JSON.stringify({
      name: $('#projectName').value,
      ownerName: $('#projectOwner').value,
      repoUrl: $('#projectRepo').value,
      defaultFormat: $('#projectFormat').value,
      environments: parseEnvironmentInput($('#projectEnvironments').value),
      description: $('#projectDescription').value
    })
  });

  await reloadProjects();
  state.activeProjectId = created.id;
  const project = activeProject();
  if (project) {
    state.activeEnvironment = project.environments.includes('prod')
      ? 'prod'
      : project.environments[0];
  }
  await loadConfigs();
  await loadDiff();
  setProjectModal(false);
  switchView('projects');
  renderAll();
  showToast(`${created.name} created`);
}

export async function rollbackLatestVersion() {
  const project = activeProject();
  const previous = state.configHistory[1];
  if (!project || !previous) {
    showToast('No previous config snapshot to restore');
    return;
  }

  const confirmed = window.confirm(
    `Rollback ${project.name} ${state.activeEnvironment} config to the previous snapshot?`
  );
  if (!confirmed) return;

  await api(`/projects/${project.id}/config-history/rollback`, {
    method: 'POST',
    body: JSON.stringify({
      environment: state.activeEnvironment,
      snapshotId: previous.id,
      changeReason: 'rollback config snapshot from frontend history'
    })
  });

  await loadConfigs();
  await loadDiff();
  await loadConfigHistory();
  renderAll();
  showToast(`${project.name} ${state.activeEnvironment} config rolled back`);
}


export async function editConfig(configId) {
  const config = state.configs.find((entry) => entry.id === configId);
  if (!config) return;

  if (isProdSensitiveEdit(config)) {
    const hasReview = await hasReviewRequest(config);
    if (!hasReview) {
      const confirmed = window.confirm(
        '此為敏感環境，是否已建立一筆 Review Request？'
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
    method: 'PUT',
    body: JSON.stringify({
      value: nextValue,
      changeReason: 'updated from frontend prototype'
    })
  });

  await loadConfigs();
  await loadDiff();
  renderAll();
  showToast(`${config.key} updated`);
}

export async function hasReviewRequest(config) {
  const requests = await api(
    `/projects/${config.projectId}/review-requests?env=prod&key=${encodeURIComponent(
      config.key
    )}&status=pending`
  );
  return requests.length > 0;
}

export async function importConfigFile() {
  const file = $('#configFile').files[0];
  const project = activeProject();
  if (!file || !project) {
    showToast('Choose a config file first');
    return;
  }

  const format = $('#configFormat').value;
  const content = await file.text();
  const result = await api(`/projects/${project.id}/configs/import`, {
    method: 'POST',
    body: JSON.stringify({
      environment: state.activeEnvironment,
      format,
      content,
      changeReason: `import ${file.name}`
    })
  });

  await loadConfigs();
  await loadDiff();
  renderAll();
  showToast(
    `Imported ${result.imported}: ${result.created} created, ${result.updated} updated`
  );
}
export function exportCurrentConfig() {
  const project = activeProject();
  if (!project) {
    showToast('Choose a project first');
    return;
  }

  const format = project.defaultFormat || 'yaml';
  const content = serializeConfigs(state.configs, format);
  const extension = format === 'properties' ? 'properties' : format === 'json' ? 'json' : 'yaml';
  downloadFile(
    `${project.name}-${state.activeEnvironment}.${extension}`,
    content,
    format === 'json' ? 'application/json' : 'text/plain'
  );
  showToast('Config file exported');
}

export function exportDiffReport() {
  const project = activeProject();
  if (!project) {
    showToast('Choose a project first');
    return;
  }

  const report = {
    projectId: project.id,
    projectName: project.name,
    generatedAt: new Date().toISOString(),
    source: 'staging',
    target: 'prod',
    differences: state.diffItems
  };
  downloadFile(
    `${project.name}-staging-prod-diff.json`,
    `${JSON.stringify(report, null, 2)}\n`,
    'application/json'
  );
  showToast('Diff report exported');
}

function serializeConfigs(configs, format) {
  const entries = [...configs].sort((a, b) => a.key.localeCompare(b.key));
  if (format === 'json') {
    return `${JSON.stringify(Object.fromEntries(entries.map((entry) => [entry.key, entry.value])), null, 2)}\n`;
  }
  if (format === 'properties') {
    return entries
      .map((entry) => `${entry.key}=${entry.value}`)
      .join('\n') + '\n';
  }
  return entries
    .map((entry) => `${entry.key}: ${JSON.stringify(entry.value)}`)
    .join('\n') + '\n';
}

function downloadFile(filename, content, type) {
  const blob = new Blob([content], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}


export async function createReviewRequest() {
  const project = activeProject();
  if (!project) return;
  const configKey = window.prompt('Config key for review request', 'database.url');
  if (configKey === null) return;
  const reason = window.prompt(
    'Reason',
    `Review ${state.activeEnvironment} change for ${project.name}`
  );
  if (!reason) return;

  await api('/review-requests', {
    method: 'POST',
    body: JSON.stringify({
      projectId: project.id,
      environment: state.activeEnvironment,
      configKey,
      reason
    })
  });

  state.requests = await api('/review-requests');
  renderDashboard();
  renderRequests();
  showToast('Review request created');
}

export async function handleReviewDecision(id, action) {
  await api(`/review-requests/${id}/${action}`, {
    method: 'PUT',
    body: JSON.stringify({ comment: `${action} from frontend` })
  });
  state.requests = await api('/review-requests');
  renderDashboard();
  renderRequests();
  showToast(`Review request ${action}d`);
}

export async function toggleSensitiveReveal(revealKey) {
  if (state.revealedKeys.has(revealKey)) {
    state.revealedKeys.delete(revealKey);
    await loadConfigs();
  } else {
    state.revealedKeys.add(revealKey);
    await loadConfigs(true);
  }
  renderConfigRows();
}
