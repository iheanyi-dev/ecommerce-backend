package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func decodeTestJSON(rec *httptest.ResponseRecorder, destination any) error {
	return json.NewDecoder(rec.Body).Decode(destination)
}

var _ = (*testing.T)(nil)
