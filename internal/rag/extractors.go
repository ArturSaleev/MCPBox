package rag

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
)

type ExtractedDocument struct {
	FilePath  string
	Fragments []ExtractedFragment
}

type ExtractedFragment struct {
	Section string
	Content string
}

func extractDocument(path string) (*ExtractedDocument, error) {
	ext := strings.ToLower(filepath.Ext(path))

	var (
		fragments []ExtractedFragment
		err       error
	)

	switch ext {
	case ".go", ".js", ".md", ".php", ".txt":
		fragments, err = extractSingleFragment(path, "", extractPlainText)
	case ".csv":
		fragments, err = extractSingleFragment(path, "Rows", extractCSV)
	case ".xlsx":
		fragments, err = extractXLSX(path)
	case ".docx":
		fragments, err = extractSingleFragment(path, "", extractDOCX)
	case ".pptx":
		fragments, err = extractPPTX(path)
	case ".pdf":
		fragments, err = extractPDF(path)
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}
	if err != nil {
		return nil, err
	}

	return &ExtractedDocument{
		FilePath:  filepath.Clean(path),
		Fragments: fragments,
	}, nil
}

func extractSingleFragment(path, section string, extractor func(string) (string, error)) ([]ExtractedFragment, error) {
	content, err := extractor(path)
	if err != nil {
		return nil, err
	}
	content = strings.TrimSpace(normalizeExtractedText(content))
	if content == "" {
		return nil, fmt.Errorf("no extractable text found")
	}
	return []ExtractedFragment{{Section: section, Content: content}}, nil
}

func extractPlainText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func extractCSV(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	var lines []string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		lines = append(lines, strings.Join(trimSlice(record), " | "))
	}

	return strings.Join(lines, "\n"), nil
}

func extractXLSX(path string) ([]ExtractedFragment, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	sharedStrings := readXLSXSharedStrings(reader.File)
	sheetNames := readXLSXSheetNames(reader.File)

	type sheetEntry struct {
		name    string
		section string
		content string
	}

	var entries []sheetEntry
	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, "xl/worksheets/") || !strings.HasSuffix(file.Name, ".xml") {
			continue
		}

		sheetContent, err := readZipFile(file)
		if err != nil {
			return nil, err
		}

		rows, err := parseXLSXSheetRows(sheetContent, sharedStrings)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			continue
		}

		sheetName := sheetNames[file.Name]
		if sheetName == "" {
			sheetName = filepath.Base(file.Name)
		}

		entries = append(entries, sheetEntry{
			name:    file.Name,
			section: "Sheet: " + sheetName,
			content: strings.Join(rows, "\n"),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	fragments := make([]ExtractedFragment, 0, len(entries))
	for _, entry := range entries {
		fragments = append(fragments, ExtractedFragment{
			Section: entry.section,
			Content: normalizeExtractedText(entry.content),
		})
	}
	if len(fragments) == 0 {
		return nil, fmt.Errorf("no extractable text found")
	}
	return fragments, nil
}

func extractDOCX(path string) (string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = reader.Close() }()

	var parts []string
	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, "word/") || !strings.HasSuffix(file.Name, ".xml") {
			continue
		}
		if file.Name == "word/styles.xml" || file.Name == "word/settings.xml" || file.Name == "word/fontTable.xml" {
			continue
		}

		data, err := readZipFile(file)
		if err != nil {
			return "", err
		}

		text := extractXMLText(bytes.NewReader(data), map[string]string{
			"t":   "",
			"tab": "\t",
			"br":  "\n",
			"cr":  "\n",
			"p":   "\n",
			"tr":  "\n",
			"tc":  "\t",
		})
		text = strings.TrimSpace(text)
		if text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, "\n\n"), nil
}

func extractPPTX(path string) ([]ExtractedFragment, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	type slideEntry struct {
		name    string
		section string
		content string
	}

	var entries []slideEntry
	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, "ppt/slides/slide") || !strings.HasSuffix(file.Name, ".xml") {
			continue
		}

		data, err := readZipFile(file)
		if err != nil {
			return nil, err
		}

		text := strings.TrimSpace(extractXMLText(bytes.NewReader(data), map[string]string{
			"t":  "",
			"br": "\n",
			"p":  "\n",
		}))
		if text == "" {
			continue
		}

		slideName := slideLabelFromPath(file.Name)
		entries = append(entries, slideEntry{
			name:    file.Name,
			section: "Slide: " + slideName,
			content: text,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	fragments := make([]ExtractedFragment, 0, len(entries))
	for _, entry := range entries {
		fragments = append(fragments, ExtractedFragment{
			Section: entry.section,
			Content: normalizeExtractedText(entry.content),
		})
	}
	if len(fragments) == 0 {
		return nil, fmt.Errorf("no extractable text found")
	}
	return fragments, nil
}

func extractPDF(path string) ([]ExtractedFragment, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var fragments []ExtractedFragment
	for pageIndex := 1; pageIndex <= reader.NumPage(); pageIndex++ {
		page := reader.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			return nil, err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		fragments = append(fragments, ExtractedFragment{
			Section: fmt.Sprintf("Page: %d", pageIndex),
			Content: text,
		})
	}

	if len(fragments) == 0 {
		return nil, fmt.Errorf("no extractable text found")
	}
	for i := range fragments {
		fragments[i].Content = normalizeExtractedText(fragments[i].Content)
	}
	return fragments, nil
}

func readZipFile(file *zip.File) ([]byte, error) {
	handle, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = handle.Close() }()

	return io.ReadAll(handle)
}

func readXLSXSharedStrings(files []*zip.File) []string {
	for _, file := range files {
		if file.Name != "xl/sharedStrings.xml" {
			continue
		}

		data, err := readZipFile(file)
		if err != nil {
			return nil
		}

		var shared struct {
			Items []struct {
				Texts []string `xml:"t"`
				Runs  []struct {
					Text string `xml:"t"`
				} `xml:"r"`
			} `xml:"si"`
		}
		if err := xml.Unmarshal(data, &shared); err != nil {
			return nil
		}

		values := make([]string, 0, len(shared.Items))
		for _, item := range shared.Items {
			if len(item.Runs) > 0 {
				var builder strings.Builder
				for _, run := range item.Runs {
					builder.WriteString(run.Text)
				}
				values = append(values, builder.String())
				continue
			}
			values = append(values, strings.Join(item.Texts, ""))
		}
		return values
	}

	return nil
}

func readXLSXSheetNames(files []*zip.File) map[string]string {
	workbookRels := map[string]string{}
	for _, file := range files {
		switch file.Name {
		case "xl/_rels/workbook.xml.rels":
			data, err := readZipFile(file)
			if err != nil {
				continue
			}
			var rels struct {
				Relationships []struct {
					ID     string `xml:"Id,attr"`
					Target string `xml:"Target,attr"`
				} `xml:"Relationship"`
			}
			if err := xml.Unmarshal(data, &rels); err != nil {
				continue
			}
			for _, rel := range rels.Relationships {
				workbookRels[rel.ID] = filepath.ToSlash(filepath.Clean(filepath.Join("xl", rel.Target)))
			}
		}
	}

	result := map[string]string{}
	for _, file := range files {
		if file.Name != "xl/workbook.xml" {
			continue
		}

		data, err := readZipFile(file)
		if err != nil {
			return result
		}

		var workbook struct {
			Sheets []struct {
				Name string `xml:"name,attr"`
				ID   string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
			} `xml:"sheets>sheet"`
		}
		if err := xml.Unmarshal(data, &workbook); err != nil {
			return result
		}

		for _, sheet := range workbook.Sheets {
			if target := workbookRels[sheet.ID]; target != "" {
				result[target] = sheet.Name
			}
		}
	}

	return result
}

func parseXLSXSheetRows(data []byte, sharedStrings []string) ([]string, error) {
	var worksheet struct {
		Rows []struct {
			Cells []struct {
				Type   string `xml:"t,attr"`
				Value  string `xml:"v"`
				Inline struct {
					Text string `xml:"t"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal(data, &worksheet); err != nil {
		return nil, err
	}

	rows := make([]string, 0, len(worksheet.Rows))
	for _, row := range worksheet.Rows {
		values := make([]string, 0, len(row.Cells))
		for _, cell := range row.Cells {
			values = append(values, resolveXLSXCellValue(cell.Type, cell.Value, cell.Inline.Text, sharedStrings))
		}
		line := strings.TrimSpace(strings.Join(trimSlice(values), " | "))
		if line != "" {
			rows = append(rows, line)
		}
	}

	return rows, nil
}

func resolveXLSXCellValue(cellType, value, inline string, sharedStrings []string) string {
	switch cellType {
	case "s":
		index, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil && index >= 0 && index < len(sharedStrings) {
			return sharedStrings[index]
		}
	case "inlineStr":
		return inline
	}
	return value
}

func extractXMLText(reader io.Reader, separators map[string]string) string {
	decoder := xml.NewDecoder(reader)

	var builder strings.Builder
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return builder.String()
		}

		switch node := token.(type) {
		case xml.StartElement:
			if separator, ok := separators[node.Name.Local]; ok && separator != "" {
				builder.WriteString(separator)
			}
		case xml.CharData:
			builder.WriteString(string(node))
		}
	}

	return builder.String()
}

func normalizeExtractedText(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	content = strings.ReplaceAll(content, "\u00a0", " ")

	repeatedBlankLines := regexp.MustCompile(`\n{3,}`)
	content = repeatedBlankLines.ReplaceAllString(content, "\n\n")

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func trimSlice(values []string) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = strings.TrimSpace(value)
	}
	return result
}

func slideLabelFromPath(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	number := strings.TrimPrefix(base, "slide")
	if number == "" {
		return base
	}
	return number
}
