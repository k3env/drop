# UI Layer Specification

## Overview

Frontend layer is built using Go templates (Echo v5 built-in renderer) with vanilla JavaScript. It provides two pages: upload and download.

## Technology Stack

- **Templates**: Go `html/template` with Echo v5 `TemplateRenderer`
- **JavaScript**: Vanilla ES6+ (no frameworks)
- **CSS**: Plain CSS with CSS variables

## Directory Structure

```
ui/
├── static/
│   ├── style.css    # Neutral minimalistic styling
│   └── app.js       # Client-side utilities and page logic
└── templates/
    ├── index.html   # Upload page (/)
    └── download.html # Download page (/:token)
```

## Pages

### 1. Upload Page (`/`)

**Purpose**: Allow users to upload files with progress tracking.

**Features**:
- Drag-and-drop file selection
- Click-to-browse file selection
- Progress bar showing upload percentage
- Error handling with user-friendly messages
- Automatic redirect to download page after successful upload

**Implementation Details**:

HTML structure:
- `.file-drop-area` - drag-and-drop zone
- `.file-input` - hidden file input element
- `.btn` - upload button (disabled until file selected)
- `.progress-container` - progress bar area

JavaScript:
- Uses `XMLHttpRequest` for upload progress tracking (not `fetch` as `fetch` doesn't support progress events)
- Event listeners: `dragover`, `dragleave`, `drop` for drag-and-drop
- On upload complete: redirect to `/{token}`

API Interaction:
- POST `/api/v1/files` with `multipart/form-data`
- Form field: `file` (required)

### 2. Download Page (`/:token`)

**Purpose**: Display file information and provide download functionality.

**Features**:
- Display filename
- Display file size (human-readable format)
- Display remaining TTL (auto-updating)
- Display file status (active/expired/consumed)
- Download button
- Shareable URL with copy-to-clipboard functionality

**Implementation Details**:

HTML structure:
- Hidden input with token value
- `.file-info` - container for file metadata
- `.btn-download` - download button link
- `.url-container` - URL display and copy button

JavaScript:
- Fetches metadata via `GET /api/v1/files/{token}`
- Updates remaining time every second via `setInterval`
- Disables download button for expired/consumed files
- Uses Clipboard API for copy functionality

API Interaction:
- GET `/api/v1/files/{token}` - fetch metadata
- GET `/{token}/download` - download file

## Data Format

### File Metadata Response

```json
{
  "token": "abc123",
  "filename": "document.pdf",
  "size": 1048576,
  "content_type": "application/pdf",
  "one_time": false,
  "expires_at": "2026-03-03T12:00:00Z",
  "status": "active"
}
```

## JavaScript Utilities

### formatBytes(bytes)

Converts bytes to human-readable format.

```javascript
formatBytes(1024)        // "1 KB"
formatBytes(1048576)     // "1 MB"
formatBytes(1073741824)  // "1 GB"
```

### formatDuration(seconds)

Converts seconds to human-readable duration.

```javascript
formatDuration(30)       // "30s"
formatDuration(3600)     // "1h 0m"
formatDuration(86400)    // "1d 0h"
```

### getRemainingTime(expiresAt)

Calculates remaining seconds until expiration.

```javascript
getRemainingTime("2026-03-03T12:00:00Z") // returns seconds remaining
```

### copyToClipboard(text, button)

Copies text to clipboard and shows feedback on button.

```javascript
copyToClipboard(url, copyButton)
// Button shows "Copied!" for 2 seconds
```

## CSS Variables

```css
:root {
    --bg-color: #f5f5f5;
    --card-bg: #ffffff;
    --text-primary: #333333;
    --text-secondary: #666666;
    --border-color: #e0e0e0;
    --accent-color: #555555;
    --accent-hover: #444444;
    --success-color: #4a9c6d;
    --error-color: #c75450;
    --radius: 8px;
    --shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}
```

## Styling Principles

- **Neutral**: No bright colors, uses grayscale with subtle accents
- **Minimal**: Clean design with adequate whitespace
- **Responsive**: Works on mobile and desktop
- **Accessible**: Clear visual feedback for interactions
- **Card-based**: Content is contained in centered cards

## Integration with Backend

The UI is integrated into the main Echo server:

```go
// main.go

// Template renderer
tmpl := template.Must(template.ParseGlob("ui/templates/*.html"))
e.Renderer = &echo.TemplateRenderer{
    Template: tmpl,
}

// Static files
e.Static("/static", "ui/static")

// UI routes (must be after API routes)
e.GET("/", handlers.Index)
e.GET("/:token", handlers.DownloadPage)
```

**Important**: API routes must be defined BEFORE UI routes to avoid routing conflicts:

```go
// API routes first
e.POST("/api/v1/files", fileHandler.Upload)
e.GET("/api/v1/files/:token", fileHandler.Metadata)
e.GET("/:token/download", fileHandler.Download)
e.DELETE("/api/v1/files/:token", fileHandler.Delete)

// Then UI routes
e.GET("/", handlers.Index)
e.GET("/:token", handlers.DownloadPage)
```

## Error Handling

- **Upload errors**: Display error message below upload button
- **Download page errors**: Show "File not found" or "File expired" message
- **Network errors**: Show generic error message, log to console

## Future Enhancements

- Add file type icons
- Add preview for images
- Add password protection option in UI
- Add file expiration configuration in UI
- Add language selection
- Add dark mode toggle
