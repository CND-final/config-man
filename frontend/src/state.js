export const API_BASE = '/api/v1';

export const state = {
  token: localStorage.getItem('configManToken') || '',
  user: JSON.parse(localStorage.getItem('configManUser') || 'null'),
  activeView: 'dashboard',
  activeProjectId: '',
  activeEnvironment: 'prod',
  configSearch: '',
  revealedKeys: new Set(),
  projects: [],
  configs: [],
  requests: [],
  templates: [],
  configHistory: [],
  historyLoading: false,
  importModalOpen: false,
  importPreview: null,
  importPreviewOpen: false,
  importApplying: false,
  projectModalOpen: false,
  historyModalOpen: false
};

export const navItems = [
  { id: 'dashboard', label: 'Dashboard', code: 'D' },
  { id: 'projects', label: 'Projects', code: 'P' },
  { id: 'templates', label: 'Templates', code: 'T' },
  { id: 'requests', label: 'Requests', code: 'R' }
];

export function activeProject() {
  return (
    state.projects.find((project) => project.id === state.activeProjectId) ||
    state.projects[0]
  );
}

export function normalizeProject(project) {
  return {
    ...project,
    owner: project.ownerName,
    environments: project.environments.map((environment) =>
      typeof environment === 'string' ? environment : environment.name
    ),
    lastChanged: 'live API',
    configCount: project.configCount ?? 0
  };
}
