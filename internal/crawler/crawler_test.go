package crawler

import (
	"mirrorer/internal/config"
	"mirrorer/internal/file"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/gocolly/colly/v2"
	"github.com/stretchr/testify/assert"
)

func listFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

var routes = map[string]struct {
	body             []byte
	status           int
	contentType      string
	redirectLocation string
}{
	"/": {
		status:      http.StatusOK,
		contentType: "text/html",
		body: []byte(`<!DOCTYPE html>
			<html>
			<head>
				<link rel="stylesheet" href="assets/style.css">
			</head>
			<body>
				<a href="/child">Visit child</a>
				<a href="/redirect">Visit redirect</a>
				<a href="/external/redirect">Visit external redirect</a>
				<img src="/assets/image.jpg">
				<script src="assets/script.js"></script>

				<a href="https://disallowed.com">Visit another domain</a>
				<a href="/disallowed">Visit another page</a>
				<a href="/404">Visit non existent page</a>
				<a href="/503">Visit broken page</a>
			</body>
			</html>`),
	},
	"/assets/style.css": {
		status:      http.StatusOK,
		contentType: "text/css",
		body: []byte(`
			@font-face {
			  font-family: 'CustomFont';
			  src: url('https://example.com/fonts/customfont.woff2') format('woff2'),
				   url('https://example.com/fonts/customfont.woff') format('woff');
			}

			body {
				background-image: url('https://example.com/images/background.jpg');
				background-size: cover;
			}

			.icon {
				background-image: url('/images/nav-icon.png');
				background-size: cover;}`),
	},
	"/assets/script.js": {
		status:      http.StatusOK,
		contentType: "text/javascript",
		body:        []byte(`console.log('Hello World');`),
	},
	"/assets/image.jpg": {
		status:      http.StatusOK,
		contentType: "image/jpeg",
		body:        []byte{0xff, 0xd8, 0xff, 0xd9},
	},
	"/child": {
		status:      http.StatusOK,
		contentType: "text/html",
		body:        []byte(`<!DOCTYPE html><html><head><title>Child</title></head></html>`),
	},
	"/disallowed": {
		status:      http.StatusOK,
		contentType: "text/html",
		body:        []byte(`<!DOCTYPE html><html><head><title>Disallowed</title></head></html>`),
	},
	"/redirect": {
		status:           http.StatusMovedPermanently,
		redirectLocation: "/redirected",
	},
	"/redirected": {
		status:      http.StatusOK,
		contentType: "text/html",
		body:        []byte(`<!DOCTYPE html><html><head><title>Redirected</title></head></html>`),
	},
	"/external/redirect": {
		status:           http.StatusSeeOther,
		redirectLocation: "https://disallowed.com",
	},
	"/404": {
		status:      http.StatusNotFound,
		contentType: "text/html",
		body:        []byte(`<!DOCTYPE html><html><head><title>404 - Not Found</title></head></html>`),
	},
	"/503": {
		status:      http.StatusServiceUnavailable,
		contentType: "text/html",
		body:        []byte(`<!DOCTYPE html><html><head><title>503 - Server Error</title></head></html>`),
	},
}

func isRedirect(status int) bool {
	return status >= 300 && status < 400
}

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()

	for path, response := range routes {
		if isRedirect(response.status) {
			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "")
				http.Redirect(w, r, response.redirectLocation, response.status)
			})
		} else {
			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", response.contentType)
				w.WriteHeader(response.status)
				w.Write(response.body)
			})
		}
	}

	srv := httptest.NewUnstartedServer(mux)
	srv.Start()
	return srv
}

func TestNewCrawler(t *testing.T) {
	cfg := &config.Config{
		UserAgent:      "custom-agent",
		AllowedDomains: "example.com",
		DisallowedURLFilters: []*regexp.Regexp{
			regexp.MustCompile(".*disallowed.*"),
		},
	}

	cr, err := NewCrawler(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, cr)
	assert.Equal(t, cfg, cr.cfg)
	assert.IsType(t, &colly.Collector{}, cr.collector)
	assert.Equal(t, "custom-agent", cr.collector.UserAgent)
	assert.Equal(t, []string{"example.com"}, cr.collector.AllowedDomains)
	assert.Equal(t, []*regexp.Regexp{regexp.MustCompile(".*disallowed.*")}, cr.collector.DisallowedURLFilters)
	assert.Equal(t, true, cr.collector.Async)
}

func TestRun(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	tests := []struct {
		name           string
		filePath       string
		expectedOutput []byte
	}{
		{
			name:           "Test index.html",
			filePath:       "/index.html",
			expectedOutput: routes["/"].body,
		},
		{
			name:           "Test style.css",
			filePath:       "/assets/style.css",
			expectedOutput: routes["/assets/style.css"].body,
		},
		{
			name:           "Test script.js",
			filePath:       "/assets/script.js",
			expectedOutput: routes["/assets/script.js"].body,
		},
		{
			name:           "Test image",
			filePath:       "/assets/image.jpg",
			expectedOutput: routes["/assets/image.jpg"].body,
		},
		{
			name:           "Test child",
			filePath:       "/child.html",
			expectedOutput: routes["/child"].body,
		},
		{
			name:           "Test redirect internal",
			filePath:       "/redirect.html",
			expectedOutput: file.RedirectHTMLBody(ts.URL + "/redirected"),
		},
		{
			name:           "Test redirected",
			filePath:       "/redirected.html",
			expectedOutput: routes["/redirected"].body,
		},
		{
			name:           "Test external redirect",
			filePath:       "/external/redirect.html",
			expectedOutput: file.RedirectHTMLBody("https://disallowed.com"),
		},
	}

	serverUrl, _ := url.Parse(ts.URL)
	hostname := serverUrl.Hostname()

	// Create a new crawler instance
	cfg := &config.Config{
		Site:           ts.URL,
		AllowedDomains: hostname,
		DisallowedURLFilters: []*regexp.Regexp{
			regexp.MustCompile("/disallowed"),
		},
	}
	cr, err := NewCrawler(cfg)
	assert.NoError(t, err)

	defer os.RemoveAll(hostname)

	cr.Run()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(hostname + tt.filePath)
			assert.NoError(t, err)
			assert.Equal(t, content, tt.expectedOutput)
		})
	}

	t.Run("correct files written", func(t *testing.T) {
		var test_paths []string
		for _, test := range tests {
			test_paths = append(test_paths, hostname+test.filePath)
		}

		files, err := listFiles(hostname)
		assert.NoError(t, err)
		assert.ElementsMatch(t, files, test_paths)
	})
}
