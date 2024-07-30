package proxy

import (
	"strings"
)

func extractNamespace(metadata string) string {
	// Extract the namespace from the metadata
	// For simplicity, assume the metadata is in the format "namespace:<namespace>"
	parts := strings.Split(metadata, ":")
	if len(parts) == 2 && parts[0] == "namespace" {
		return parts[1]
	}
	return "public" // Default namespace
}
