package rag

import (
	"os"
	"path/filepath"
	"strings"
)

func ResolveManagedSourcePath(baseRoot, collectionID string) string {
	root := strings.TrimSpace(baseRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			root = "."
		} else {
			root = cwd
		}
	}
	return filepath.Join(root, "knowledge_base", "managed", sanitizePathSegment(collectionID))
}

func ResolveIndexPath(baseRoot, collectionID string) string {
	root := strings.TrimSpace(baseRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			root = "."
		} else {
			root = cwd
		}
	}
	return filepath.Join(root, "knowledge_base", "indexes", sanitizePathSegment(collectionID))
}

func sanitizePathSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "collection"
	}

	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-", " ", "-")
	return replacer.Replace(value)
}
