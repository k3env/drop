export interface UploadResponse {
  token: string;
  expires_at: string | null;
}

export interface FileMetadataResponse {
  token: string;
  filename: string;
  size: number;
  content_type: string;
  one_time: boolean;
  expires_at: string | null;
  status: FileStatus;
}

export type FileStatus = 'active' | 'expired' | 'consumed' | 'deleted';

export interface ErrorResponse {
  error: string;
  message: string;
}

export interface UploadOptions {
  ttl_seconds: number | null;
  one_time: boolean;
}
