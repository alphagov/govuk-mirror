package file

import (
	"errors"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedirectHTMLBody(t *testing.T) {
	redirectURL := "https://example.com"
	expectedOutput := []byte(`<!DOCTYPE html>
	<html lang="en">
	<head>
	<meta http-equiv="refresh" content="1; url=https://example.com">
	<title>Redirecting</title>
	</head>
	<body>
	<p>Redirecting you to <a href="https://example.com">https://example.com</a>.</p>
	</body>
	</html>`)
	output := RedirectHTMLBody(redirectURL)
	if string(output) != string(expectedOutput) {
		t.Errorf("expected %s, got %s", expectedOutput, output)
	}
}

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

func TestFindCssUrls(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected []string
	}{
		{
			name:     "basic",
			input:    []byte(`body { background: url("/image.png"); }`),
			expected: []string{"/image.png"},
		},
		{
			name:     "multiple urls",
			input:    []byte(`body { background: url("/image.png"); color: url('/colors.css'); font: url(/font.woff); }`),
			expected: []string{"/image.png", "/colors.css", "/font.woff"},
		},
		{
			name:     "no urls",
			input:    []byte(`body { color: red; }`),
			expected: []string{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			actual := FindCssUrls(tt.input)
			assert.Equal(t, actual, tt.expected)
		})
	}
}
