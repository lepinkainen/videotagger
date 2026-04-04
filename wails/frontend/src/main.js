import './style.css';

const backend = window.go?.main?.App;

const state = {
  groups: [],
  totalSelectedCount: 0,
  groupsWithSelections: {},
  selectedGroupIndex: 0,
  selectedFileIndex: 0,
  preview: null,
  previewLoading: false
};

const elements = {
  directoryInput: document.getElementById('directoryInput'),
  browseBtn: document.getElementById('browseBtn'),
  scanBtn: document.getElementById('scanBtn'),
  statusText: document.getElementById('statusText'),
  groupCount: document.getElementById('groupCount'),
  selectedCount: document.getElementById('selectedCount'),
  groupList: document.getElementById('groupList'),
  groupTitle: document.getElementById('groupTitle'),
  groupMeta: document.getElementById('groupMeta'),
  fileList: document.getElementById('fileList'),
  selectAllBtn: document.getElementById('selectAllBtn'),
  clearBtn: document.getElementById('clearBtn'),
  deleteBtn: document.getElementById('deleteBtn'),
  previewPane: document.getElementById('previewPane'),
  previewMeta: document.getElementById('previewMeta'),
  previewHint: document.getElementById('previewHint')
};

function setStatus(message, isError = false) {
  elements.statusText.textContent = message;
  elements.statusText.style.color = isError ? '#b23a27' : '';
}

function applyState(appState) {
  state.groups = appState.groups || [];
  state.totalSelectedCount = appState.totalSelectedCount || 0;
  state.groupsWithSelections = appState.groupsWithSelections || {};

  if (state.selectedGroupIndex >= state.groups.length) {
    state.selectedGroupIndex = Math.max(state.groups.length - 1, 0);
  }

  const currentGroup = state.groups[state.selectedGroupIndex];
  if (!currentGroup || state.selectedFileIndex >= currentGroup.files.length) {
    state.selectedFileIndex = 0;
  }
}

function currentGroup() {
  return state.groups[state.selectedGroupIndex];
}

function currentFile() {
  const group = currentGroup();
  if (!group) {
    return null;
  }
  return group.files[state.selectedFileIndex] || null;
}

function formatFileSize(bytes) {
  const kb = 1024;
  const mb = kb * 1024;
  const gb = mb * 1024;
  if (bytes >= gb) {
    return `${(bytes / gb).toFixed(1)} GB`;
  }
  if (bytes >= mb) {
    return `${(bytes / mb).toFixed(1)} MB`;
  }
  if (bytes >= kb) {
    return `${(bytes / kb).toFixed(1)} KB`;
  }
  return `${bytes} B`;
}

function formatModTime(value) {
  if (!value) {
    return 'unknown date';
  }
  const numeric = typeof value === 'number' ? value : Number(value);
  const msValue = Number.isNaN(numeric) ? value : (numeric < 1e12 ? numeric * 1000 : numeric);
  const date = new Date(msValue);
  if (Number.isNaN(date.getTime())) {
    return 'unknown date';
  }
  return date.toLocaleString();
}

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;');
}

function renderGroups() {
  elements.groupList.innerHTML = '';
  state.groups.forEach((group, index) => {
    const selectedCount = group.selected?.filter(Boolean).length || 0;
    const card = document.createElement('div');
    card.className = `group-item${index === state.selectedGroupIndex ? ' active' : ''}`;
    card.dataset.groupIndex = `${index}`;
    card.style.animationDelay = `${index * 30}ms`;
    card.innerHTML = `
      <strong>${escapeHtml(group.hash)}</strong>
      <span>${group.files.length} files - ${selectedCount} selected</span>
    `;
    elements.groupList.appendChild(card);
  });
}

function renderFiles() {
  elements.fileList.innerHTML = '';
  const group = currentGroup();
  if (!group) {
    elements.groupTitle.textContent = 'Files';
    elements.groupMeta.textContent = 'Select a group to begin.';
    return;
  }

  elements.groupTitle.textContent = `Group ${state.selectedGroupIndex + 1}`;
  elements.groupMeta.textContent = `Hash ${group.hash} - ${group.files.length} files`;

  group.files.forEach((file, index) => {
    const baseName = file.path.split(/[/\\]/).pop();
    const metadata = [
      file.resolution || '???',
      file.durationMins ? `${file.durationMins}min` : '?min',
      formatFileSize(file.size || 0),
      formatModTime(file.modTime)
    ].join(' | ');

    const card = document.createElement('div');
    const selectedClass = group.selected?.[index] ? ' selected' : '';
    const activeClass = index === state.selectedFileIndex ? ' active' : '';
    card.className = `file-card${selectedClass}${activeClass}`;
    card.dataset.fileIndex = `${index}`;
    card.style.animationDelay = `${index * 20}ms`;
    card.innerHTML = `
      <button class="checkbox${group.selected?.[index] ? ' selected' : ''}" data-action="toggle" data-file-index="${index}" type="button">X</button>
      <div class="file-info">
        <h3>${escapeHtml(baseName)}</h3>
        <p>${escapeHtml(file.path)}</p>
        <p>${metadata}</p>
      </div>
    `;
    elements.fileList.appendChild(card);
  });
}

function renderPreview() {
  const file = currentFile();
  if (!file) {
    elements.previewMeta.textContent = 'No file selected.';
    elements.previewPane.innerHTML = '<div class="preview-placeholder">Pick a file to see it here.</div>';
    elements.previewHint.textContent = '';
    return;
  }

  elements.previewMeta.textContent = file.path;

  if (state.previewLoading) {
    elements.previewPane.innerHTML = '<div class="preview-placeholder">Generating preview...</div>';
    elements.previewHint.textContent = '';
    return;
  }

  if (!state.preview) {
    elements.previewPane.innerHTML = '<div class="preview-placeholder">Preview unavailable.</div>';
    elements.previewHint.textContent = '';
    return;
  }

  if (state.preview.type === 'image') {
    elements.previewPane.innerHTML = `<img src="${state.preview.data}" alt="Preview" />`;
    elements.previewHint.textContent = 'Frame preview generated by ffmpeg.';
    return;
  }

  if (state.preview.type === 'video') {
    elements.previewPane.innerHTML = `
      <video controls src="${state.preview.data}"></video>
    `;
    elements.previewHint.textContent = state.preview.error || 'Video playback preview.';
    return;
  }

  elements.previewPane.innerHTML = '<div class="preview-placeholder">Preview unavailable.</div>';
  elements.previewHint.textContent = state.preview.error || '';
}

function renderStats() {
  elements.groupCount.textContent = `${state.groups.length} groups`;
  elements.selectedCount.textContent = `${state.totalSelectedCount} selected`;
}

function updateActions() {
  const hasGroups = state.groups.length > 0;
  elements.selectAllBtn.disabled = !hasGroups;
  elements.clearBtn.disabled = !hasGroups;
  elements.deleteBtn.disabled = state.totalSelectedCount === 0;

  document.querySelectorAll('[data-strategy]').forEach((button) => {
    button.disabled = !hasGroups;
  });
}

function renderAll() {
  renderGroups();
  renderFiles();
  renderPreview();
  renderStats();
  updateActions();
}

async function scanDirectory() {
  if (!backend) {
    setStatus('Backend not available.', true);
    return;
  }

  const directory = elements.directoryInput.value.trim();
  if (!directory) {
    setStatus('Enter a folder to scan.', true);
    return;
  }

  setStatus(`Scanning ${directory}...`);
  try {
    const appState = await backend.ScanDirectory(directory);
    applyState(appState);
    setStatus(state.groups.length ? 'Duplicates loaded.' : 'No duplicates found.');
    await loadPreview();
    renderAll();
  } catch (error) {
    setStatus(`Scan failed: ${error}`, true);
  }
}

async function loadPreview() {
  if (!backend) {
    return;
  }

  const file = currentFile();
  if (!file) {
    state.preview = null;
    state.previewLoading = false;
    return;
  }

  state.previewLoading = true;
  renderPreview();
  try {
    state.preview = await backend.GetPreview(file.path);
  } catch (error) {
    state.preview = { type: 'error', error: String(error) };
  } finally {
    state.previewLoading = false;
  }
}

async function runGroupAction(action, ...args) {
  if (!backend) {
    return;
  }

  try {
    const appState = await backend[action](...args);
    applyState(appState);
    await loadPreview();
    renderAll();
  } catch (error) {
    setStatus(String(error), true);
  }
}

async function toggleSelection(index) {
  await runGroupAction('ToggleSelection', state.selectedGroupIndex, index);
}

async function applyAutoSelect(strategy) {
  await runGroupAction('ApplyAutoSelect', state.selectedGroupIndex, strategy);
}

async function deleteSelected() {
  if (!backend) {
    return;
  }

  const selectedCount = state.totalSelectedCount;
  if (!selectedCount) {
    setStatus('No files selected.', true);
    return;
  }

  const confirmed = window.confirm(`Delete ${selectedCount} file(s)? This cannot be undone.`);
  if (!confirmed) {
    return;
  }

  setStatus('Deleting selected files...');
  await runGroupAction('DeleteSelected');
  setStatus('Deletion complete.');
}

function bindEvents() {
  elements.scanBtn.addEventListener('click', scanDirectory);
  elements.browseBtn.addEventListener('click', async () => {
    if (!backend) {
      return;
    }
    try {
      const directory = await backend.SelectDirectory();
      if (directory) {
        elements.directoryInput.value = directory;
      }
    } catch (error) {
      setStatus(String(error), true);
    }
  });

  elements.groupList.addEventListener('click', async (event) => {
    const item = event.target.closest('.group-item');
    if (!item) {
      return;
    }
    const index = Number(item.dataset.groupIndex);
    if (Number.isNaN(index)) {
      return;
    }
    state.selectedGroupIndex = index;
    state.selectedFileIndex = 0;
    await loadPreview();
    renderAll();
  });

  elements.fileList.addEventListener('click', async (event) => {
    const toggle = event.target.closest('[data-action="toggle"]');
    if (toggle) {
      const index = Number(toggle.dataset.fileIndex);
      if (!Number.isNaN(index)) {
        await toggleSelection(index);
      }
      return;
    }

    const card = event.target.closest('.file-card');
    if (!card) {
      return;
    }
    const index = Number(card.dataset.fileIndex);
    if (Number.isNaN(index)) {
      return;
    }
    state.selectedFileIndex = index;
    await loadPreview();
    renderAll();
  });

  elements.selectAllBtn.addEventListener('click', () => runGroupAction('SelectAllInGroup', state.selectedGroupIndex));
  elements.clearBtn.addEventListener('click', () => runGroupAction('ClearSelectionInGroup', state.selectedGroupIndex));
  elements.deleteBtn.addEventListener('click', deleteSelected);

  document.querySelectorAll('[data-strategy]').forEach((button) => {
    button.addEventListener('click', () => {
      const strategy = Number(button.dataset.strategy);
      if (!Number.isNaN(strategy)) {
        applyAutoSelect(strategy);
      }
    });
  });
}

function init() {
  if (!backend) {
    setStatus('Wails backend not detected. Run with Wails for full functionality.', true);
  }
  bindEvents();
  renderAll();
}

init();
