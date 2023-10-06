package file

import (
	"errors"
	"net/url"
	"os"
	"testing"
)

func TestSave(t *testing.T) {
	u, _ := url.Parse("https://example.com/foo/bar")
	contentType := "text/html"
	body := []byte("Hello, World!")

	err := Save(u, contentType, body)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	content, err := os.ReadFile("example.com/foo/bar.html")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if string(content) != string(body) {
		t.Errorf("file to contain %v, got %v", body, content)
	}
	defer os.RemoveAll("example.com")
}

func TestGenerateFilePathTableDriven(t *testing.T) {
	var tests = []struct {
		url, contentType string
		want             string
		wantErr          error
	}{
		{"https://example.com", "text/html", "example.com/index.html", nil},
		{"https://example.com/", "text/html", "example.com/index.html", nil},
		{"https://example.com/foo", "text/html", "example.com/foo.html", nil},
		{"https://example.com/foo/", "text/html", "example.com/foo/index.html", nil},
		{"https://example.com/foo/bar", "text/html", "example.com/foo/bar.html", nil},
		{"https://example.com/foo", "text/csv", "example.com/foo.csv", nil},
		{"https://example.com/style.css", "text/css", "example.com/style.css", nil},
		{"https://example.com/foo", "text/blah", "", errors.New("Error determining content type")},
		{"https://example.com/foo.cy", "text/html", "example.com/foo.cy.html", nil},
		{"https://example.com/foo.html", "text/html", "example.com/foo.html", nil},
		{"https://example.com/foo//bar", "text/html", "example.com/foo/bar.html", nil},
		{"https://example.com/foo?a=b&c=d", "text/html", "example.com/foo.html", nil},
		{"https://example.com/foo?a=b&c=d#hello", "text/html", "example.com/foo.html", nil},
		{"https://example.com/foo#hello", "text/html", "example.com/foo.html", nil},
		{"https://example.com/foo%20bar", "text/html", "example.com/foo%20bar.html", nil},
		{"https://example.com/foo", "", "", errors.New("mime: no media type")},
	}

	for _, tt := range tests {
		u, _ := url.Parse(tt.url)
		output, err := GenerateFilePath(u, tt.contentType)
		if (err != nil) != (tt.wantErr != nil) || (err != nil && err.Error() != tt.wantErr.Error()) {
			t.Errorf("GenerateFilePath() error = %v, wantErr %v", err, tt.wantErr)
		}
		if output != tt.want {
			t.Errorf("got %s, want %s", output, tt.want)
		}
	}
}
