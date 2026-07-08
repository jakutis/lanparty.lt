package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// representationRequest is the entity template for POST /v1/representation.
type representationRequest struct {
	Type string `json:"type"` // the kind of file to generate, e.g. "html"
	Spec string `json:"spec"` // a natural-language prompt describing the file
}

// representationHandler handles POST /v1/representation. It generates a file
// from the given spec and returns it as the response entity, using the
// requested type to determine the content type and download filename.
func representationHandler(gen Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("representation: received request from %s", r.RemoteAddr)

		var req representationRequest
		if err := decodeJSON(r, &req); err != nil {
			log.Printf("representation: rejecting body: %v", err)
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}
		req.Type = strings.TrimSpace(req.Type)
		req.Spec = strings.TrimSpace(req.Spec)
		log.Printf("representation: parsed request type=%q spec=%q", req.Type, req.Spec)
		if req.Type == "" || req.Spec == "" {
			log.Printf("representation: rejecting request: missing required fields")
			writeError(w, http.StatusUnprocessableEntity, "fields 'type' and 'spec' are required")
			return
		}

		ct, ext, ok := contentTypeFor(req.Type)
		if !ok {
			log.Printf("representation: rejecting request: unsupported type %q", req.Type)
			writeError(w, http.StatusUnprocessableEntity, "unsupported type "+strconv.Quote(req.Type)+": only \"html\" and \"pdf\" are supported")
			return
		}

		content, err := gen.Generate(r.Context(), req.Type, req.Spec)
		if err != nil {
			log.Printf("representation: generation failed: %v", err)
			writeError(w, http.StatusInternalServerError, "generation failed: "+err.Error())
			return
		}
		log.Printf("representation: generated %d bytes for type=%q", len(content), req.Type)

		log.Printf("representation: responding with content-type=%q filename=%q", ct, "representation"+ext)
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Content-Disposition", `attachment; filename="representation`+ext+`"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}
}

// decodeJSON decodes a single JSON object from the request body into dst,
// rejecting unknown fields, oversized bodies and trailing content.
func decodeJSON(r *http.Request, dst any) error {
	const maxBody = 1 << 20 // 1 MiB
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBody+1))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeError(w http.ResponseWriter, code int, msg string) {
	log.Printf("representation: writing %d error response: %s", code, msg)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
