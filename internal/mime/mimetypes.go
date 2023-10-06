package mime

import (
	"fmt"
	"mime"
)

var additionalMimeTypes = map[string]string{
	".atom": "application/atom+xml",
	".csv":  "text/csv",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".ico":  "image/x-icon",
	".ics":  "text/calendar",
	".odp":  "application/vnd.oasis.opendocument.presentation",
	".ods":  "application/vnd.oasis.opendocument.spreadsheet",
	".odt":  "application/vnd.oasis.opendocument.text",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
}

func LoadAdditionalMimeTypes() error {
	for ext, typ := range additionalMimeTypes {
		if err := mime.AddExtensionType(ext, typ); err != nil {
			return fmt.Errorf("error adding mime type %s with extension %s: %w", typ, ext, err)
		}
	}
	return nil
}
