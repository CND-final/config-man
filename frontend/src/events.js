import {
  createProject,
  createReviewRequest,
  editConfig,
  exportCurrentConfig,
  handleReviewDecision,
  extractConfigFile,
  applyImportPreview,
  openVersionHistory,
  rollbackLatestVersion,
  setHistoryModal,
  setImportModal,
  setImportPreviewModal,
  setProjectModal,
  switchView,
  toggleSensitiveReveal
} from './actions.js';
import { api } from './api.js';
import { loadConfigsAndHistory, loadInitialData } from './data.js';
import { $, setAuthenticated, showToast } from './dom.js';
import { renderAll, renderConfigRows } from './render.js';
import { activeProject, API_BASE, state } from './state.js';

export function bindEvents() {
  document.addEventListener('click', handleDocumentClick);

  $('#loginForm').addEventListener('submit', handleLogin);
  $('#logoutButton').addEventListener('click', handleLogout);

  $('#configSearch').addEventListener('input', (event) => {
    state.configSearch = event.target.value;
    renderConfigRows();
  });

  $('#configFile').addEventListener('change', (event) => {
    const name = event.target.files[0]?.name || '';
    const ext = name.split('.').pop()?.toLowerCase();
    if (ext === 'json') $('#configFormat').value = 'json';
    if (ext === 'yaml' || ext === 'yml') $('#configFormat').value = 'yaml';
    if (ext === 'properties') $('#configFormat').value = 'properties';
  });

  $('#importConfig').addEventListener('click', () => {
    extractConfigFile().catch((error) => showToast(error.message));
  });

  $('#submitReview').addEventListener('click', () => {
    createReviewRequest().catch((error) => showToast(error.message));
  });

  $('#exportConfig').addEventListener('click', () => {
    exportCurrentConfig();
  });

  $('#projectForm').addEventListener('submit', (event) => {
    createProject(event).catch((error) => showToast(error.message));
  });
}

export function initApp() {
  $('#apiBaseLabel').textContent = API_BASE;
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
  const target = event.target.closest('button');
  if (!target) return;

  try {
    if (target.dataset.closeModal === 'project') {
      setProjectModal(false);
      return;
    }

    if (target.dataset.closeModal === 'history') {
      setHistoryModal(false);
      return;
    }

    if (target.dataset.closeModal === 'import') {
      setImportModal(false);
      return;
    }

    if (target.dataset.closeModal === 'import-preview') {
      setImportPreviewModal(false);
      return;
    }

    if (target.id === 'registerProject') {
      setProjectModal(true);
      return;
    }

    if (target.id === 'openImportConfig') {
      setImportModal(true);
      return;
    }

    if (target.id === 'rollbackLatest') {
      await rollbackLatestVersion();
      return;
    }

    if (target.id === 'applyImportConfig') {
      await applyImportPreview();
      return;
    }

    const jump = target.dataset.jump;
    if (jump) {
      switchView(jump);
      if (target.dataset.openProjectForm) {
        setProjectModal(true);
      }
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
      state.activeEnvironment = project.environments.includes('prod')
        ? 'prod'
        : project.environments[0];
      await loadConfigsAndHistory();
      switchView('config');
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

    if (target.id === 'openConfigHistory') {
      await openVersionHistory();
      return;
    }

    if (target.dataset.editConfig) {
      await editConfig(target.dataset.editConfig);
      return;
    }

    if (target.dataset.approve) {
      await handleReviewDecision(target.dataset.approve, 'approve');
      return;
    }

    if (target.dataset.reject) {
      await handleReviewDecision(target.dataset.reject, 'reject');
    }
  } catch (error) {
    showToast(error.message);
  }
}

async function handleLogin(event) {
  event.preventDefault();
  try {
    const data = await api('/auth/login', {
      method: 'POST',
      body: JSON.stringify({
        email: $('#loginEmail').value,
        password: $('#loginPassword').value
      })
    });
    state.token = data.token;
    state.user = data.user;
    localStorage.setItem('configManToken', state.token);
    localStorage.setItem('configManUser', JSON.stringify(state.user));
    setAuthenticated(true);
    await loadInitialData();
    renderAll();
    showToast(`Signed in as ${state.user.name}`);
  } catch (error) {
    showToast(error.message);
  }
}

function handleLogout() {
  localStorage.removeItem('configManToken');
  localStorage.removeItem('configManUser');
  state.token = '';
  state.user = null;
  setAuthenticated(false);
}
