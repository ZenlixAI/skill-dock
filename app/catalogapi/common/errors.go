package common

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"skill_registry/business/catalog"
)

// CatalogErrorFromError normalizes an arbitrary error into a *catalog.Error,
// passing catalog errors through and mapping anything else to a 500.
func CatalogErrorFromError(err error) *catalog.Error {
	if err == nil {
		return nil
	}
	if catalogErr, ok := err.(*catalog.Error); ok {
		return catalogErr
	}
	return catalog.NewError(500, "internal_error", "Internal server error.", nil)
}

func StatusFromError(err error) int {
	catalogErr, ok := err.(*catalog.Error)
	if !ok {
		return http.StatusInternalServerError
	}
	return catalogErr.Status
}

func HashRequestPayload(payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
