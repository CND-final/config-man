import { api } from './api.js';
import { activeProject, normalizeProject, state } from './state.js';

export async function loadInitialData() {
  const [projects, template, requests] = await Promise.all([
    api('/projects'),
    api('/templates/base'),
    api('/review-requests')
  ]);

  state.projects = projects.map(normalizeProject);
  state.templates = [
    {
      name: template.name,
      format: 'base',
      description: 'Required baseline keys from the backend template.',
      keys: template.entries.map((entry) => entry.key)
    }
  ];
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

export async function reloadProjects() {
  const projects = await api('/projects');
  state.projects = projects.map(normalizeProject);
}

export async function loadConfigs(revealSensitive = false) {
  const project = activeProject();
  if (!project) {
    state.configs = [];
    state.configHistory = [];
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
