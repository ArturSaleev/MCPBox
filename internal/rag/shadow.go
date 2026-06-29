package rag

import (
	"fmt"
	"path/filepath"
	"strings"
)

func ExtractDocument(path string) (*ExtractedDocument, error) {
	return extractDocument(path)
}

func RenderExtractedDocumentMarkdown(document *ExtractedDocument) string {
	if document == nil {
		return ""
	}

	var builder strings.Builder
	fileName := filepath.Base(strings.TrimSpace(document.FilePath))
	if fileName != "" {
		builder.WriteString("# ")
		builder.WriteString(fileName)
		builder.WriteString("\n\n")
	}

	for index, fragment := range document.Fragments {
		section := strings.TrimSpace(fragment.Section)
		if section != "" {
			builder.WriteString("## ")
			builder.WriteString(section)
			builder.WriteString("\n\n")
		} else if index > 0 {
			builder.WriteString("## Fragment ")
			builder.WriteString(fmt.Sprintf("%d", index+1))
			builder.WriteString("\n\n")
		}

		content := strings.TrimSpace(fragment.Content)
		if content == "" {
			continue
		}
		builder.WriteString(content)
		builder.WriteString("\n\n")
	}

	return strings.TrimSpace(builder.String()) + "\n"
}
