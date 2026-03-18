function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function formatDuration(seconds) {
    if (seconds <= 0) return 'Expired';
    
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    
    if (days > 0) {
        return `${days}d ${hours}h`;
    } else if (hours > 0) {
        return `${hours}h ${minutes}m`;
    } else if (minutes > 0) {
        return `${minutes}m`;
    } else {
        return `${seconds}s`;
    }
}

function getRemainingTime(expiresAt) {
    const expires = new Date(expiresAt).getTime();
    const now = Date.now();
    const remaining = Math.max(0, Math.floor((expires - now) / 1000));
    return remaining;
}

async function copyToClipboard(text, button) {
    try {
        await navigator.clipboard.writeText(text);
        const originalText = button.textContent;
        button.textContent = 'Copied!';
        button.style.backgroundColor = 'var(--success-color)';
        setTimeout(() => {
            button.textContent = originalText;
            button.style.backgroundColor = '';
        }, 2000);
    } catch (err) {
        console.error('Failed to copy:', err);
    }
}

function initUploadPage() {
    const dropArea = document.getElementById('dropArea');
    const fileInput = document.getElementById('fileInput');
    const fileName = document.getElementById('fileName');
    const uploadBtn = document.getElementById('uploadBtn');
    const progressContainer = document.getElementById('progressContainer');
    const progressFill = document.getElementById('progressFill');
    const progressText = document.getElementById('progressText');
    const errorMessage = document.getElementById('errorMessage');
    const ttlSelect = document.getElementById('ttlSelect');
    const noExpiry = document.getElementById('noExpiry');
    const oneTime = document.getElementById('oneTime');

    if (!dropArea) return;

    oneTime.addEventListener('change', () => {
        if (oneTime.checked) {
            ttlSelect.disabled = true;
            noExpiry.checked = false;
        } else {
            ttlSelect.disabled = false;
        }
    });

    noExpiry.addEventListener('change', () => {
        if (noExpiry.checked) {
            ttlSelect.disabled = true;
            oneTime.checked = false;
        } else {
            ttlSelect.disabled = false;
        }
    });

    dropArea.addEventListener('click', () => fileInput.click());

    dropArea.addEventListener('dragover', (e) => {
        e.preventDefault();
        dropArea.classList.add('dragover');
    });

    dropArea.addEventListener('dragleave', () => {
        dropArea.classList.remove('dragover');
    });

    dropArea.addEventListener('drop', (e) => {
        e.preventDefault();
        dropArea.classList.remove('dragover');
        const files = e.dataTransfer.files;
        if (files.length > 0) {
            fileInput.files = files;
            updateFileName(files[0].name);
        }
    });

    fileInput.addEventListener('change', () => {
        if (fileInput.files.length > 0) {
            updateFileName(fileInput.files[0].name);
        }
    });

    function updateFileName(name) {
        fileName.textContent = name;
        fileName.style.display = 'block';
        uploadBtn.disabled = false;
        errorMessage.classList.remove('visible');
    }

    uploadBtn.addEventListener('click', async () => {
        if (!fileInput.files.length) return;

        const file = fileInput.files[0];
        const formData = new FormData();
        formData.append('file', file);

        if (oneTime.checked) {
            formData.append('one_time', 'true');
            formData.append('ttl_seconds', '0');
        } else if (noExpiry.checked) {
            formData.append('ttl_seconds', '0');
        } else {
            formData.append('ttl_seconds', ttlSelect.value);
        }

        uploadBtn.disabled = true;
        uploadBtn.textContent = 'Uploading...';
        errorMessage.classList.remove('visible');
        progressContainer.classList.add('visible');
        progressFill.style.width = '0%';
        progressText.textContent = '0%';

        try {
            const xhr = new XMLHttpRequest();

            await new Promise((resolve, reject) => {
                xhr.upload.addEventListener('progress', (e) => {
                    if (e.lengthComputable) {
                        const percent = Math.round((e.loaded / e.total) * 100);
                        progressFill.style.width = percent + '%';
                        progressText.textContent = percent + '%';
                    }
                });

                xhr.addEventListener('load', () => {
                    if (xhr.status >= 200 && xhr.status < 300) {
                        resolve(xhr.response);
                    } else {
                        reject(new Error(xhr.responseText));
                    }
                });

                xhr.addEventListener('error', reject);
                xhr.addEventListener('abort', reject);

                xhr.open('POST', '/api/v1/files');
                xhr.send(formData);
            });

            const response = JSON.parse(xhr.responseText);
            window.location.href = '/' + response.token;

        } catch (err) {
            errorMessage.textContent = 'Upload failed. Please try again.';
            errorMessage.classList.add('visible');
            uploadBtn.disabled = false;
            uploadBtn.textContent = 'Upload File';
            progressContainer.classList.remove('visible');
        }
    });
}

async function initDownloadPage() {
    const token = document.getElementById('token').value;
    const fileInfo = document.getElementById('fileInfo');
    const downloadBtn = document.getElementById('downloadBtn');
    const copyBtn = document.getElementById('copyBtn');
    const urlInput = document.getElementById('urlInput');
    const expiresAtEl = document.getElementById('expiresAt');
    const statusEl = document.getElementById('status');

    if (!token) return;

    try {
        const response = await fetch(`/api/v1/files/${token}`);
        
        if (!response.ok) {
            if (response.status === 404) {
                fileInfo.innerHTML = '<p style="text-align:center;color:var(--error-color);">File not found</p>';
                downloadBtn.disabled = true;
                return;
            }
            if (response.status === 410) {
                fileInfo.innerHTML = '<p style="text-align:center;color:var(--error-color);">File has expired</p>';
                downloadBtn.disabled = true;
                return;
            }
            throw new Error('Failed to fetch file info');
        }

        const data = await response.json();

        document.getElementById('filename').textContent = data.filename;
        document.getElementById('filesize').textContent = formatBytes(data.size);
        
        if (data.one_time) {
            document.getElementById('oneTimeRow').style.display = 'flex';
        }
        
        if (data.expires_at === null) {
            expiresAtEl.textContent = 'No expiration';
        } else {
            const remaining = getRemainingTime(data.expires_at);
            expiresAtEl.textContent = formatDuration(remaining);
        }
        
        statusEl.textContent = data.status;
        statusEl.className = 'file-status ' + data.status;

        if (data.status === 'expired' || data.status === 'consumed') {
            downloadBtn.disabled = true;
            downloadBtn.textContent = data.status === 'expired' ? 'File Expired' : 'Already Downloaded';
        }

        if (data.expires_at !== null) {
            const updateTimer = () => {
                const remaining = getRemainingTime(data.expires_at);
                expiresAtEl.textContent = formatDuration(remaining);
                
                if (remaining <= 0 && data.status === 'active') {
                    statusEl.textContent = 'expired';
                    statusEl.className = 'file-status expired';
                    downloadBtn.disabled = true;
                    downloadBtn.textContent = 'File Expired';
                }
            };

            setInterval(updateTimer, 1000);
        }

    } catch (err) {
        console.error('Error:', err);
        fileInfo.innerHTML = '<p style="text-align:center;color:var(--error-color);">Error loading file info</p>';
        downloadBtn.disabled = true;
    }

    if (copyBtn && urlInput) {
        copyBtn.addEventListener('click', () => {
            copyToClipboard(urlInput.value, copyBtn);
        });
    }
}

document.addEventListener('DOMContentLoaded', () => {
    initUploadPage();
    initDownloadPage();
});
