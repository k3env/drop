import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import type { FileMetadataResponse } from '../api/types';
import { formatBytes, formatDuration, getRemainingTime, copyToClipboard } from '../utils';

export function DownloadPage() {
  const { token } = useParams<{ token: string }>();
  const [metadata, setMetadata] = useState<FileMetadataResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [remainingTime, setRemainingTime] = useState<number | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!token) return;

    fetch(`/api/v1/files/${token}`)
      .then((res) => {
        if (!res.ok) {
          if (res.status === 404) {
            throw new Error('File not found');
          }
          if (res.status === 410) {
            throw new Error('File has expired');
          }
          throw new Error('Failed to fetch file info');
        }
        return res.json();
      })
      .then((data: FileMetadataResponse) => {
        setMetadata(data);
        if (data.expires_at) {
          setRemainingTime(getRemainingTime(data.expires_at));
        }
      })
      .catch((err: Error) => {
        setError(err.message);
      });
  }, [token]);

  useEffect(() => {
    if (!metadata?.expires_at || metadata.status !== 'active') return;

    const interval = setInterval(() => {
      const remaining = getRemainingTime(metadata.expires_at!);
      setRemainingTime(remaining);

      if (remaining <= 0) {
        setMetadata((prev) => prev ? { ...prev, status: 'expired' } : null);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [metadata?.expires_at, metadata?.status]);

  const handleCopy = async () => {
    if (!token) return;
    const url = `${window.location.origin}/${token}`;
    await copyToClipboard(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (error) {
    return (
      <div className="container">
        <div className="card">
          <p className="error-message visible" style={{ textAlign: 'center' }}>
            {error}
          </p>
          <div className="footer">
            <Link to="/">Upload another file</Link>
          </div>
        </div>
      </div>
    );
  }

  if (!metadata) {
    return (
      <div className="container">
        <div className="card">
          <p style={{ textAlign: 'center', color: 'var(--text-secondary)' }}>
            Loading...
          </p>
        </div>
      </div>
    );
  }

  const isDisabled = metadata.status === 'expired' || metadata.status === 'consumed';

  return (
    <div className="container">
      <div className="card">
        <h2>{metadata.filename}</h2>
        <p className="subtitle">Ready to download</p>

        <div className="file-info">
          <div className="file-info-item">
            <span className="file-info-label">Size</span>
            <span className="file-info-value">{formatBytes(metadata.size)}</span>
          </div>
          {metadata.one_time && (
            <div className="file-info-item">
              <span className="file-info-label">Type</span>
              <span className="file-info-value" style={{ color: 'var(--accent-color)' }}>
                One-time download
              </span>
            </div>
          )}
          <div className="file-info-item">
            <span className="file-info-label">Expires in</span>
            <span className="file-info-value">
              {metadata.expires_at
                ? formatDuration(remainingTime ?? 0)
                : 'No expiration'}
            </span>
          </div>
          <div className="file-info-item">
            <span className="file-info-label">Status</span>
            <span className={`file-status ${metadata.status}`}>
              {metadata.status}
            </span>
          </div>
        </div>

        <a
          href={`/${token}/download`}
          className="btn btn-download"
          style={{ display: 'block', textAlign: 'center', textDecoration: 'none' }}
          onClick={(e) => isDisabled && e.preventDefault()}
        >
          {isDisabled
            ? metadata.status === 'expired'
              ? 'File Expired'
              : 'Already Downloaded'
            : 'Download File'}
        </a>

        <div className="url-container">
          <input
            type="text"
            className="url-input"
            value={`${window.location.origin}/${token}`}
            readOnly
          />
          <button
            className="btn btn-copy btn-secondary"
            onClick={handleCopy}
          >
            {copied ? 'Copied!' : 'Copy'}
          </button>
        </div>

        <div className="footer">
          <Link to="/">Upload another file</Link>
        </div>
      </div>
    </div>
  );
}
