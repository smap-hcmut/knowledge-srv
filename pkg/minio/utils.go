package minio

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

// generateDownloadHeaders generates HTTP headers for file download responses.
func (m *implMinIO) generateDownloadHeaders(objInfo minio.ObjectInfo, req *DownloadRequest) *DownloadHeaders {
	disposition := m.determineContentDisposition(objInfo.ContentType, req.Disposition)
	originalName := objInfo.UserMetadata["original-name"]
	if originalName == "" {
		originalName = objInfo.Key
	}

	headers := &DownloadHeaders{
		ContentType:        objInfo.ContentType,
		ContentDisposition: fmt.Sprintf("%s; filename=\"%s\"", disposition, originalName),
		ContentLength:      fmt.Sprintf("%d", objInfo.Size),
		LastModified:       objInfo.LastModified.Format(http.TimeFormat),
		ETag:               objInfo.ETag,
		AcceptRanges:       "bytes",
	}

	if disposition == "inline" {
		headers.CacheControl = "public, max-age=3600"
	} else {
		headers.CacheControl = "private, no-cache"
	}

	return headers
}

// determineContentDisposition determines the appropriate content disposition based on content type and request.
func (m *implMinIO) determineContentDisposition(contentType, requestedDisposition string) string {
	if requestedDisposition == "inline" || requestedDisposition == "attachment" {
		return requestedDisposition
	}

	if requestedDisposition == "auto" {
		viewableTypes := []string{
			"image/", "video/", "audio/",
			"application/pdf", "text/plain", "text/html",
			"application/json", "application/xml",
		}

		for _, viewable := range viewableTypes {
			if strings.HasPrefix(contentType, viewable) {
				return "inline"
			}
		}
		return "attachment"
	}

	return "attachment"
}

// generateObjectName generates a unique object name based on the original filename and prefix.
func (m *implMinIO) generateObjectName(originalName, prefix string) string {
	if originalName == "" {
		return fmt.Sprintf("%s/%d", prefix, time.Now().UnixNano())
	}

	// Clean the filename
	cleanName := filepath.Base(originalName)
	cleanName = strings.ReplaceAll(cleanName, " ", "_")
	cleanName = strings.ReplaceAll(cleanName, "\\", "_")
	cleanName = strings.ReplaceAll(cleanName, "/", "_")

	// Add timestamp for uniqueness
	timestamp := time.Now().Format("20060102_150405")
	ext := filepath.Ext(cleanName)
	nameWithoutExt := strings.TrimSuffix(cleanName, ext)

	return fmt.Sprintf("%s/%s_%s%s", prefix, nameWithoutExt, timestamp, ext)
}

// getContentTypeFromFilename determines the MIME content type from the file extension.
func (m *implMinIO) getContentTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".webp": "image/webp",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".html": "text/html",
		".htm":  "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".wmv":  "video/x-ms-wmv",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	}

	if contentType, exists := contentTypes[ext]; exists {
		return contentType
	}

	return "application/octet-stream"
}

// formatFileSize formats file size in human-readable format (e.g., "1.5 MB").
func (m *implMinIO) formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// isImageFile checks if the file is an image based on its content type.
func (m *implMinIO) isImageFile(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// isVideoFile checks if the file is a video based on its content type.
func (m *implMinIO) isVideoFile(contentType string) bool {
	return strings.HasPrefix(contentType, "video/")
}

// isAudioFile checks if the file is an audio file based on its content type.
func (m *implMinIO) isAudioFile(contentType string) bool {
	return strings.HasPrefix(contentType, "audio/")
}

// isDocumentFile checks if the file is a document based on its content type.
func (m *implMinIO) isDocumentFile(contentType string) bool {
	documentTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text/plain",
		"text/html",
	}

	for _, docType := range documentTypes {
		if contentType == docType {
			return true
		}
	}
	return false
}

// sanitizeMetadata cleans and validates metadata by removing empty values, normalizing keys, and limiting value length.
func (m *implMinIO) sanitizeMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return make(map[string]string)
	}

	sanitized := make(map[string]string)
	for key, value := range metadata {
		// Remove empty values
		if value == "" {
			continue
		}

		// Sanitize key (lowercase, replace spaces with underscores)
		sanitizedKey := strings.ToLower(strings.ReplaceAll(key, " ", "_"))

		// Limit value length
		if len(value) > 1024 {
			value = value[:1024]
		}

		sanitized[sanitizedKey] = value
	}

	return sanitized
}

// buildObjectPath builds a proper object path from multiple path components, removing leading/trailing slashes.
func (m *implMinIO) buildObjectPath(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			// Remove leading/trailing slashes
			cleanPart := strings.Trim(part, "/")
			if cleanPart != "" {
				cleanParts = append(cleanParts, cleanPart)
			}
		}
	}
	return strings.Join(cleanParts, "/")
}
