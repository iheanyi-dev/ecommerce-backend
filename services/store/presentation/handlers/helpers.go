package handlers

import (
	"encoding/json"
	"io"
	"net/http"
)

// decodeJSON decodes exactly one JSON value from the request body.
//
// Keeping this at the HTTP boundary prevents malformed request payloads from
// crossing into the application layer.
func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	// Reject trailing JSON values instead of silently accepting a body
	// containing multiple independent JSON documents.
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return err
	}

	return nil
}
