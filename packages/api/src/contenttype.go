package main

import (
	"mime"
	"strings"
)

// contentTypeFor maps a logical file type (the request "type" field) to the
// HTTP Content-Type to return and a file extension (including the leading dot)
// used for the download filename.
func contentTypeFor(typ string) (contentType, extension string) {
	switch strings.ToLower(typ) {
	case "html", "htm":
		return "text/html; charset=utf-8", ".html"
	case "css":
		return "text/css; charset=utf-8", ".css"
	case "js", "javascript":
		return "text/javascript; charset=utf-8", ".js"
	case "json":
		return "application/json; charset=utf-8", ".json"
	case "csv":
		return "text/csv; charset=utf-8", ".csv"
	case "txt", "text":
		return "text/plain; charset=utf-8", ".txt"
	case "md", "markdown":
		return "text/markdown; charset=utf-8", ".md"
	case "xml":
		return "application/xml; charset=utf-8", ".xml"
	case "svg":
		return "image/svg+xml", ".svg"
	case "png":
		return "image/png", ".png"
	case "jpg", "jpeg":
		return "image/jpeg", ".jpg"
	default:
		if ct := mime.TypeByExtension("." + strings.ToLower(typ)); ct != "" {
			return ct, "." + strings.ToLower(typ)
		}
		return "application/octet-stream", "." + typ
	}
}
