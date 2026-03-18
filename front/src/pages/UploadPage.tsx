import { useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import type { UploadResponse } from '../api/types';

export function UploadPage() {
  const navigate = useNavigate();
  const [file, setFile] = useState<File | null>(null);
  const [ttl, setTtl] = useState<number>(86400);
  const [noExpiry, setNoExpiry] = useState(false);
  const [oneTime, setOneTime] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    const files = e.dataTransfer.files;
    if (files.length > 0) {
      setFile(files[0]);
      setError(null);
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files && files.length > 0) {
      setFile(files[0]);
      setError(null);
    }
  };

  const handleUpload = async () => {
    if (!file) return;

    setUploading(true);
    setError(null);
    setProgress(0);

    const formData = new FormData();
    formData.append('file', file);

    if (oneTime) {
      formData.append('one_time', 'true');
      formData.append('ttl_seconds', '0');
    } else if (noExpiry) {
      formData.append('ttl_seconds', '0');
    } else {
      formData.append('ttl_seconds', ttl.toString());
    }

    return new Promise<void>((resolve, reject) => {
      const xhr = new XMLHttpRequest();

      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable) {
          setProgress(Math.round((e.loaded / e.total) * 100));
        }
      });

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          const response: UploadResponse = JSON.parse(xhr.responseText);
          navigate(`/${response.token}`);
          resolve();
        } else {
          let errorMsg = 'Upload failed. Please try again.';
          try {
            const errResp = JSON.parse(xhr.responseText);
            if (errResp.message) errorMsg = errResp.message;
          } catch {}
          setError(errorMsg);
          setUploading(false);
          reject(new Error(xhr.responseText));
        }
      });

      xhr.addEventListener('error', () => {
        setError('Upload failed. Please try again.');
        setUploading(false);
        reject(new Error('Network error'));
      });

      xhr.addEventListener('abort', () => {
        setUploading(false);
        reject(new Error('Aborted'));
      });

      xhr.open('POST', '/api/v1/files');
      xhr.send(formData);
    });
  };

  return (
    <div className="container">
      <div className="card">
        <h1>Drop File</h1>
        <p className="subtitle">Share files instantly</p>

        <div
          className="file-drop-area"
          onClick={() => fileInputRef.current?.click()}
          onDragOver={(e) => e.preventDefault()}
          onDragLeave={() => {}}
          onDrop={handleDrop}
        >
          <input
            ref={fileInputRef}
            type="file"
            className="file-input"
            onChange={handleFileChange}
          />
          {!file ? (
            <label className="file-label">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="17 8 12 3 7 8" />
                <line x1="12" y1="3" x2="12" y2="15" />
              </svg>
              <span>Drop file here or click to browse</span>
            </label>
          ) : (
            <span className="file-name">{file.name}</span>
          )}
        </div>

        {file && (
          <div className="options-container">
            <div className="option-row">
              <label htmlFor="ttlSelect">Expiration:</label>
              <select
                id="ttlSelect"
                className="ttl-select"
                value={ttl}
                onChange={(e) => setTtl(Number(e.target.value))}
                disabled={noExpiry || oneTime}
              >
                <option value={3600}>1 hour</option>
                <option value={86400}>24 hours</option>
                <option value={604800}>7 days</option>
              </select>
            </div>
            <div className="option-row">
              <label className="checkbox-label">
                <input
                  type="checkbox"
                  checked={noExpiry}
                  onChange={(e) => {
                    setNoExpiry(e.target.checked);
                    if (e.target.checked) setOneTime(false);
                  }}
                />
                <span>No expiration</span>
              </label>
            </div>
            <div className="option-row">
              <label className="checkbox-label">
                <input
                  type="checkbox"
                  checked={oneTime}
                  onChange={(e) => {
                    setOneTime(e.target.checked);
                    if (e.target.checked) setNoExpiry(false);
                  }}
                />
                <span>One-time download</span>
              </label>
            </div>
          </div>
        )}

        <button
          className="btn"
          onClick={handleUpload}
          disabled={!file || uploading}
        >
          {uploading ? 'Uploading...' : 'Upload File'}
        </button>

        {uploading && (
          <div className="progress-container visible">
            <div className="progress-bar">
              <div className="progress-fill" style={{ width: `${progress}%` }} />
            </div>
            <p className="progress-text">{progress}%</p>
          </div>
        )}

        {error && (
          <div className="error-message visible">{error}</div>
        )}
      </div>
    </div>
  );
}
