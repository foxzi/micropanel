// File Manager JavaScript
let currentPath = '/';
let currentEditPath = null;
let siteID = null;
let csrfToken = null;

function initFileManager(id, token) {
    siteID = id;
    csrfToken = token;
    loadFiles('/');
}

async function loadFiles(path) {
    currentPath = path;
    document.getElementById('current-path').textContent = path;
    document.getElementById('upload-path').value = path;

    try {
        const response = await fetch(`/sites/${siteID}/files?path=${encodeURIComponent(path)}`);
        const data = await response.json();

        if (!response.ok) {
            alert(data.error || 'Failed to load files');
            return;
        }

        renderFileTree(data.files || [], path);
    } catch (error) {
        console.error('Error loading files:', error);
    }
}

function renderFileTree(files, path) {
    const container = document.getElementById('file-tree');

    let html = '';

    // Parent directory link
    if (path !== '/') {
        const parentPath = path.split('/').slice(0, -1).join('/') || '/';
        html += `<div class="py-1 px-2 hover:bg-gray-100 cursor-pointer flex items-center" onclick="loadFiles('${parentPath}')">
            <span class="mr-2">📁</span>
            <span class="text-blue-600">..</span>
        </div>`;
    }

    // Sort: directories first, then files
    files.sort((a, b) => {
        if (a.is_dir && !b.is_dir) return -1;
        if (!a.is_dir && b.is_dir) return 1;
        return a.name.localeCompare(b.name);
    });

    for (const file of files) {
        const icon = file.is_dir ? '📁' : '📄';
        const clickAction = file.is_dir
            ? `loadFiles('${file.path}')`
            : `openFile('${file.path}')`;

        html += `<div class="py-1 px-2 hover:bg-gray-100 cursor-pointer flex items-center justify-between group">
            <div class="flex items-center flex-1" onclick="${clickAction}">
                <span class="mr-2">${icon}</span>
                <span class="${file.is_dir ? 'text-blue-600' : ''}">${file.name}</span>
            </div>
            <div class="hidden group-hover:flex space-x-1">
                <button onclick="renameItem('${file.path}', '${file.name}')" class="text-blue-500 hover:text-blue-700 text-xs">Rename</button>
                <button onclick="deleteItem('${file.path}')" class="text-red-500 hover:text-red-700 text-xs">Delete</button>
                ${!file.is_dir ? `<button onclick="downloadFile('${file.path}')" class="text-green-500 hover:text-green-700 text-xs">Download</button>` : ''}
            </div>
        </div>`;
    }

    if (files.length === 0 && path === '/') {
        html = '<div class="text-gray-500 text-center py-4">No files yet</div>';
    }

    container.innerHTML = html;
}

async function openFile(path) {
    try {
        const response = await fetch(`/sites/${siteID}/files/read?path=${encodeURIComponent(path)}`);
        const data = await response.json();

        if (!response.ok) {
            alert(data.error || 'Failed to read file');
            return;
        }

        if (data.is_image) {
            showPreview(path);
        } else if (data.is_text) {
            showEditor(path, data.content);
        } else {
            if (confirm('This file type cannot be edited. Download instead?')) {
                downloadFile(path);
            }
        }
    } catch (error) {
        console.error('Error reading file:', error);
    }
}

function showEditor(path, content) {
    currentEditPath = path;
    document.getElementById('editor-filename').textContent = path;
    document.getElementById('editor').value = content;
    document.getElementById('editor-container').classList.remove('hidden');
    document.getElementById('preview-container').classList.add('hidden');
    document.getElementById('placeholder').classList.add('hidden');
}

function showPreview(path) {
    document.getElementById('preview-filename').textContent = path;
    document.getElementById('preview-image').src = `/sites/${siteID}/files/preview?path=${encodeURIComponent(path)}`;
    document.getElementById('preview-container').classList.remove('hidden');
    document.getElementById('editor-container').classList.add('hidden');
    document.getElementById('placeholder').classList.add('hidden');
}

function closeEditor() {
    currentEditPath = null;
    document.getElementById('editor-container').classList.add('hidden');
    document.getElementById('placeholder').classList.remove('hidden');
}

function closePreview() {
    document.getElementById('preview-container').classList.add('hidden');
    document.getElementById('placeholder').classList.remove('hidden');
}

async function saveFile() {
    if (!currentEditPath) return;

    const content = document.getElementById('editor').value;

    try {
        const response = await fetch(`/sites/${siteID}/files/write`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': csrfToken,
            },
            body: JSON.stringify({
                path: currentEditPath,
                content: content,
            }),
        });

        const data = await response.json();

        if (!response.ok) {
            alert(data.error || 'Failed to save file');
            return;
        }

        alert('File saved!');
    } catch (error) {
        console.error('Error saving file:', error);
    }
}

function createFile() {
    document.getElementById('create-modal-title').textContent = 'New File';
    document.getElementById('create-is-dir').value = 'false';
    document.getElementById('create-name').value = '';
    document.getElementById('create-name').placeholder = 'filename.html';
    document.getElementById('create-modal').classList.remove('hidden');
}

function createFolder() {
    document.getElementById('create-modal-title').textContent = 'New Folder';
    document.getElementById('create-is-dir').value = 'true';
    document.getElementById('create-name').value = '';
    document.getElementById('create-name').placeholder = 'folder-name';
    document.getElementById('create-modal').classList.remove('hidden');
}

function closeCreateModal() {
    document.getElementById('create-modal').classList.add('hidden');
}

async function submitCreate(e) {
    e.preventDefault();

    const name = document.getElementById('create-name').value;
    const isDir = document.getElementById('create-is-dir').value === 'true';
    const path = currentPath === '/' ? `/${name}` : `${currentPath}/${name}`;

    try {
        const response = await fetch(`/sites/${siteID}/files/create`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': csrfToken,
            },
            body: JSON.stringify({
                path: path,
                is_dir: isDir,
            }),
        });

        const data = await response.json();

        if (!response.ok) {
            alert(data.error || 'Failed to create');
            return;
        }

        closeCreateModal();
        loadFiles(currentPath);
    } catch (error) {
        console.error('Error creating:', error);
    }
}

function uploadFile() {
    document.getElementById('upload-modal').classList.remove('hidden');
}

function closeUploadModal() {
    document.getElementById('upload-modal').classList.add('hidden');
}

async function submitUpload(e) {
    e.preventDefault();

    const formData = new FormData(e.target);

    try {
        const response = await fetch(`/sites/${siteID}/files/upload`, {
            method: 'POST',
            headers: {
                'X-CSRF-Token': csrfToken,
            },
            body: formData,
        });

        const data = await response.json();

        if (!response.ok) {
            alert(data.error || 'Failed to upload');
            return;
        }

        closeUploadModal();
        loadFiles(currentPath);
    } catch (error) {
        console.error('Error uploading:', error);
    }
}

function renameItem(path, oldName) {
    document.getElementById('rename-old-path').value = path;
    document.getElementById('rename-new-name').value = oldName;
    document.getElementById('rename-modal').classList.remove('hidden');
}

function closeRenameModal() {
    document.getElementById('rename-modal').classList.add('hidden');
}

async function submitRename(e) {
    e.preventDefault();

    const oldPath = document.getElementById('rename-old-path').value;
    const newName = document.getElementById('rename-new-name').value;
    const parentPath = oldPath.split('/').slice(0, -1).join('/') || '/';
    const newPath = parentPath === '/' ? `/${newName}` : `${parentPath}/${newName}`;

    try {
        const response = await fetch(`/sites/${siteID}/files/rename`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': csrfToken,
            },
            body: JSON.stringify({
                old_path: oldPath,
                new_path: newPath,
            }),
        });

        const data = await response.json();

        if (!response.ok) {
            alert(data.error || 'Failed to rename');
            return;
        }

        closeRenameModal();
        loadFiles(currentPath);
    } catch (error) {
        console.error('Error renaming:', error);
    }
}

async function deleteItem(path) {
    if (!confirm(`Delete ${path}?`)) return;

    try {
        const response = await fetch(`/sites/${siteID}/files?path=${encodeURIComponent(path)}`, {
            method: 'DELETE',
            headers: {
                'X-CSRF-Token': csrfToken,
            },
        });

        const data = await response.json();

        if (!response.ok) {
            alert(data.error || 'Failed to delete');
            return;
        }

        loadFiles(currentPath);
    } catch (error) {
        console.error('Error deleting:', error);
    }
}

function downloadFile(path) {
    window.location.href = `/sites/${siteID}/files/download?path=${encodeURIComponent(path)}`;
}

// Bind form events when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    const createForm = document.getElementById('create-form');
    if (createForm) {
        createForm.addEventListener('submit', submitCreate);
    }

    const uploadForm = document.getElementById('upload-form');
    if (uploadForm) {
        uploadForm.addEventListener('submit', submitUpload);
    }

    const renameForm = document.getElementById('rename-form');
    if (renameForm) {
        renameForm.addEventListener('submit', submitRename);
    }
});
