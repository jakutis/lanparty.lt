package main

import (
	"strings"
)

// contentTypeFor maps a supported logical file type (the request "type" field,
// matched case-insensitively) to the HTTP Content-Type to return and a file
// extension (including the leading dot) used for the download filename. Only
// "html" and "pdf" are supported; ok reports whether the type is one of them.
func contentTypeFor(typ string) (contentType, extension string, ok bool) {
	switch strings.ToLower(typ) {
	case "html":
		return "text/html; charset=utf-8", ".html", true
	case "pdf":
		return "application/pdf", ".pdf", true
	default:
		return "", "", false
	}
}
