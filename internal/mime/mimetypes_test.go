package mime

import (
	"mime"
	"testing"
)

func TestLoadAdditionalMimeTypes(t *testing.T) {
	if err := LoadAdditionalMimeTypes(); err != nil {
		t.Errorf("LoadAdditionalMimeTypes() got error = %v", err)
	}

	ext := ".csv"
	want := "text/csv; charset=utf-8"
	got := mime.TypeByExtension(ext)
	if got != want {
		t.Errorf("TypeByExtension(%v) = %v; want %v", ext, got, want)
	}
}
