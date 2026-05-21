import { api } from './api.js';
import { activeProject, normalizeProject, state } from './state.js';

export async function loadInitialData() {
  const [projects, templates, requests] = await Promise.all([
    api('/projects'),
    api('/templates'),
    api('/review-requests')
  ]);

  state.projects = projects.map(normalizeProject);
  state.templates = normalizeTemplates(templates);
  state.requests = requests;

  if (!state.activeProjectId && state.projects[0]) {
    state.activeProjectId = state.projects[0].id;
  }

  const project = activeProject();
  if (project && !project.environments.includes(state.activeEnvironment)) {
    state.activeEnvironment = project.environments[0];
  }

  await loadConfigsAndHistory();
}

function normalizeTemplates(templates) {
  return templates.map((template) => ({
    ...template,
    format: template.format || 'base',
    keys: template.entries?.map((entry) => entry.key) || [],
    variables: template.variables || []
  }));
}

export async function reloadProjects() {
  const projects = await api('/projects');
  state.projects = projects.map(normalizeProject);
}

export async function reloadTemplates() {
  const templates = await api('/templates');
  state.templates = normalizeTemplates(templates);
}

export async function loadConfigs(revealSensitive = false) {
  const project = activeProject();
  if (!project) {
    state.configs = [];
    state.configHistory = [];
    state.configBaseline = new Map();
    state.configBaselineContext = '';
    state.pendingReviewChanges = [];
    return;
  }

  const data = await api(
    `/projects/${project.id}/configs?env=${encodeURIComponent(
      state.activeEnvironment
    )}${revealSensitive ? '&revealSensitive=true' : ''}`
  );
  state.configs = data.entries.map((entry) => ({
    ...entry,
    updated: entry.updatedBy
  }));

  const context = `${project.id}:${state.activeEnvironment}`;
  if (state.configBaselineContext !== context) {
    state.configBaselineContext = context;
    state.configBaseline = new Map(
      state.configs.map((entry) => [entry.id, baselineConfig(entry)])
    );
    state.pendingReviewChanges = [];
  }
}

function baselineConfig(entry) {
  return {
    id: entry.id,
    key: entry.key,
    value: entry.value,
    environment: entry.environment
  };
}

export async function loadConfigsAndHistory(revealSensitive = false) {
  await Promise.all([loadConfigs(revealSensitive), loadConfigHistory()]);
}

export async function loadConfigHistory() {
  const project = activeProject();
  if (!project) {
    state.configHistory = [];
    return;
  }

  const data = await api(
    `/projects/${project.id}/config-history?env=${encodeURIComponent(state.activeEnvironment)}`
  );
  state.configHistory = data.snapshots || [];
}
