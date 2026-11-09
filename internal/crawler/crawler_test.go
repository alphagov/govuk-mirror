package crawler

import (
	"context"
	"fmt"
	"mirrorer/internal/config"
	"mirrorer/internal/file"
	"mirrorer/internal/metrics"
	"mirrorer/internal/mime"
	"mirrorer/internal/upload/uploadfakes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/gocolly/colly/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

// The fixture includes generatedSitemaps extra sitemaps, each listing
// generatedPagesPerSitemap pages. With ASYNC on, as in production, these are
// fetched and parsed in parallel, so the crawl exercises the concurrent
// handling of sitemap entries.
const (
	generatedSitemaps        = 20
	generatedPagesPerSitemap = 50
)

func generatedSitemapPath(i int) string {
	return fmt.Sprintf("/generated_sitemap_%d.xml", i)
}

func generatedPagePath(sitemap, page int) string {
	return fmt.Sprintf("/generated/%d/%d", sitemap, page)
}

// sitemapIndexBody lists the hand-written sitemaps followed by the generated ones.
func sitemapIndexBody() []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>/sitemap_1.xml</loc>
		<lastmod>2023-10-10T02:50:02+00:00</lastmod>
	</sitemap>
	<sitemap>
		<loc>/sitemap_2.xml</loc>
		<lastmod>2025-11-06T00:00:00+00:00</lastmod>
	</sitemap>
`)
	for i := range generatedSitemaps {
		fmt.Fprintf(&b, "\t<sitemap><loc>%s</loc></sitemap>\n", generatedSitemapPath(i))
	}
	b.WriteString(`</sitemapindex>`)
	return []byte(b.String())
}

func generatedSitemapBody(i int) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
`)
	for j := range generatedPagesPerSitemap {
		fmt.Fprintf(&b, "\t<url><loc>%s</loc><lastmod>2024-01-%02dT00:00:00+00:00</lastmod></url>\n",
			generatedPagePath(i, j), j%28+1)
	}
	b.WriteString(`</urlset>`)
	return []byte(b.String())
}

func init() {
	for i := range generatedSitemaps {
		routes[generatedSitemapPath(i)] = route{
			status:      http.StatusOK,
			contentType: "application/xml",
			body:        generatedSitemapBody(i),
		}
		for j := range generatedPagesPerSitemap {
			routes[generatedPagePath(i, j)] = route{
				status:      http.StatusOK,
				contentType: "text/html",
				body:        []byte(`<!DOCTYPE html><html><head><title>Generated</title></head></html>`),
			}
		}
	}
}

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

type route struct {
	body             []byte
	status           int
	contentType      string
	redirectLocation string
}

var routes = map[string]route{
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
				<a href="/spreadsheet.xlsx">Spreadsheet</a>
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
	"/sitemap.xml": {
		status:      http.StatusOK,
		contentType: "application/xml",
		body:        sitemapIndexBody(),
	},
	"/sitemap_1.xml": {
		status:      http.StatusOK,
		contentType: "application/xml",
		body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
					  <urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
						<url>
						  <loc>/</loc>
						  <lastmod>2025-11-06T11:00:00+00:00</lastmod>
						  <priority>0.5</priority>
						</url>
						</urlset>`),
	},
	"/sitemap_2.xml": {
		status:      http.StatusOK,
		contentType: "application/xml",
		body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
					  <urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
						<url>
						  <loc>/1</loc>
						  <lastmod>2025-11-05T11:00:00+00:00</lastmod>
						  <priority>0.5</priority>
						</url>
						<url>
						  <loc>/2</loc>
						  <lastmod>2025-11-07T11:00:00+00:00</lastmod>
						  <priority>0.5</priority>
						</url>
						<url>
						  <loc>/3</loc>
						  <priority>0.5</priority>
						</url>
						<url>
						  <loc>/500</loc>
						  <lastmod>2025-01-07T11:00:00+00:00</lastmod>
						  <priority>0.5</priority>
						</url>
						</urlset>`),
	},
	"/assets/style.css": {
		status:      http.StatusOK,
		contentType: "text/css",
		body: []byte(`
			@font-face {
			  font-family: 'CustomFont';
			  src: url('https://example.com/fonts/customfont.woff2') format('woff2');
			}

			.icon {
				background-image: url('/assets/background.png');
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
	"/assets/background.png": {
		status:      http.StatusOK,
		contentType: "image/png",
		body:        []byte{0xff, 0xd8, 0xff, 0xd9},
	},
	"/spreadsheet.xlsx": {
		status:      http.StatusOK,
		contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		body:        []byte{0x50, 0x4b, 0x03, 0x04, 0x14, 0x00, 0x06, 0x00, 0x08, 0x00, 0x00, 0x00, 0x21, 0x00, 0x36, 0x9d},
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
	"/500": {
		status:      http.StatusInternalServerError,
		contentType: "text/html",
		body:        []byte(`<!DOCTYPE html><html><head><title>500 - Server Error</title></head></html>`),
	},
	"/503": {
		status:      http.StatusServiceUnavailable,
		contentType: "text/html",
		body:        []byte(`<!DOCTYPE html><html><head><title>503 - Server Error</title></head></html>`),
	},
	"/1": {
		status:      http.StatusOK,
		contentType: "text/html",
		body:        []byte(`<!DOCTYPE html><html><head><title>1</title></head></html>`),
	},
	"/2": {
		status:      http.StatusOK,
		contentType: "text/html",
		body:        []byte(`<!DOCTYPE html><html><head><title>2</title></head></html>`),
	},
	"/3": {
		status:      http.StatusOK,
		contentType: "text/html",
		body:        []byte(`<!DOCTYPE html><html><head><title>3</title></head></html>`),
	},
}

func isRedirect(status int) bool {
	return status >= 300 && status < 400
}

// testServer serves routes and records which paths were requested. Requests
// arrive on concurrent goroutines, so the record is guarded by a mutex.
type testServer struct {
	*httptest.Server
	mu      sync.Mutex
	visited map[string]bool
}

func (s *testServer) recordVisit(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.visited[path] = true
}

func (s *testServer) wasVisited(path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.visited[path]
}

func newTestServer(t *testing.T) *testServer {
	s := &testServer{visited: map[string]bool{}}
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
				_, err := w.Write(response.body)
				if err != nil {
					t.Error("Test server unable to write response")
				}
				s.recordVisit(r.URL.Path)
			})
		}
	}

	s.Server = httptest.NewServer(mux)
	return s
}

func TestNewCrawler(t *testing.T) {
	// Setup Config for Crawler
	cfg := &config.Config{
		UserAgent:      "custom-agent",
		AllowedDomains: []string{"example.com"},
		URLFilters: []*regexp.Regexp{
			regexp.MustCompile(".*"),
		},
		DisallowedURLFilters: []*regexp.Regexp{
			regexp.MustCompile(".*disallowed.*"),
		},
		Async:              false,
		MirrorS3BucketName: "s3-bucket-name",
	}

	// Create a registry
	reg := prometheus.NewRegistry()

	// Initialize metrics
	m := metrics.NewMetrics(reg)

	// Create Crawler instance
	cr, err := NewCrawler(cfg, m, nil)

	// Assertions on Crawler instances
	assert.Nil(t, err)
	assert.NotNil(t, cr)
	assert.Equal(t, cfg, cr.cfg)
	assert.IsType(t, &colly.Collector{}, cr.collector)
	assert.Equal(t, "custom-agent", cr.collector.UserAgent)
	assert.Equal(t, []string{"example.com"}, cr.collector.AllowedDomains)
	assert.Equal(t, []*regexp.Regexp{regexp.MustCompile(".*")}, cr.collector.URLFilters)
	assert.Equal(t, []*regexp.Regexp{regexp.MustCompile(".*disallowed.*")}, cr.collector.DisallowedURLFilters)
}

func TestRun(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	tests := []struct {
		name           string
		filePath       string
		expectedOutput []byte
	}{
		{
			name:           "Test sitemap index",
			filePath:       "/sitemap.xml",
			expectedOutput: routes["/sitemap.xml"].body,
		},
		{
			name:           "Test sitemap 1",
			filePath:       "/sitemap_1.xml",
			expectedOutput: routes["/sitemap_1.xml"].body,
		},
		{
			name:           "Test sitemap 2",
			filePath:       "/sitemap_2.xml",
			expectedOutput: routes["/sitemap_2.xml"].body,
		},
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
			name:           "Test css image",
			filePath:       "/assets/background.png",
			expectedOutput: routes["/assets/background.png"].body,
		},
		{
			name:           "Test spreadsheet",
			filePath:       "/spreadsheet.xlsx",
			expectedOutput: routes["/spreadsheet.xlsx"].body,
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
		{
			name:           "Test 1",
			filePath:       "/1.html",
			expectedOutput: routes["/1"].body,
		},
		{
			name:           "Test 2",
			filePath:       "/2.html",
			expectedOutput: routes["/2"].body,
		},
		{
			name:           "Test 3",
			filePath:       "/3.html",
			expectedOutput: routes["/3"].body,
		},
	}

	serverUrl, _ := url.Parse(ts.URL)
	hostname := serverUrl.Hostname()

	// Every generated sitemap and page should be saved and uploaded, on top of
	// the hand-written files in tests.
	var generatedFiles []string
	for i := range generatedSitemaps {
		generatedFiles = append(generatedFiles, hostname+generatedSitemapPath(i))
		for j := range generatedPagesPerSitemap {
			generatedFiles = append(generatedFiles, hostname+generatedPagePath(i, j)+".html")
		}
	}
	expectedFiles := len(tests) + len(generatedFiles)

	// Make mime types across consistent different systems
	err := mime.LoadAdditionalMimeTypes()
	if err != nil {
		t.Fatalf("could not load mimetypes: %v", err)
	}

	// Create a new crawler instance
	cfg := &config.Config{
		Site:           ts.URL + "/sitemap.xml",
		Async:          true,
		Concurrency:    10,
		AllowedDomains: []string{hostname},
		URLFilters: []*regexp.Regexp{
			regexp.MustCompile(".*"),
		},
		DisallowedURLFilters: []*regexp.Regexp{
			regexp.MustCompile("/disallowed"),
		},
		MirrorS3BucketName: "s3-bucket-name",
	}

	// Create a registry
	reg := prometheus.NewRegistry()

	// Initialize metrics
	m := metrics.NewMetrics(reg)

	// Initialize uploader
	uploader := &uploadfakes.FakeUploader{}
	uploader.UploadFileStub = func(ctx context.Context, file string, key string, contentType string) error {
		if file == hostname+"/3.html" {
			return fmt.Errorf("error uploading")
		} else {
			return nil
		}
	}

	// Create a Crawler instance
	cr, err := NewCrawler(cfg, m, uploader)
	assert.NoError(t, err)

	defer func() {
		if err := os.RemoveAll(hostname); err != nil {
			fmt.Println("Error when removing:", err)
		}
	}()

	// Run the Crawler
	cr.Run(m, reg, cfg)

	// Assert that the errorCounter metric has been incremented twice for 404 and 503 errors
	t.Run("correct errorCounter metric", func(t *testing.T) {
		assert.Equal(t, float64(3), testutil.ToFloat64(m.HttpErrorCounter()))
	})

	/*
		Assert that the downloadCounter metric has been incremented by the number of
		files in the test array
	*/
	t.Run("correct downloadCounter metric", func(t *testing.T) {
		assert.Equal(t, float64(expectedFiles), testutil.ToFloat64(m.DownloadCounter()))
	})

	/*
		Assert that the crawledPagesCounter metric has been incremented by the number of
		files in the test array
	*/
	t.Run("correct crawledPagesCounter metric", func(t *testing.T) {
		assert.Equal(t, float64(expectedFiles), testutil.ToFloat64(m.CrawledPagesCounter()))
	})

	// Assert that the expected content matches the actual content saved
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(hostname + tt.filePath)
			assert.NoError(t, err)
			assert.Equal(t, content, tt.expectedOutput)
		})
	}

	// Assert that the correct files have been written
	t.Run("correct files written", func(t *testing.T) {
		var test_paths []string
		for _, test := range tests {
			test_paths = append(test_paths, hostname+test.filePath)
		}
		test_paths = append(test_paths, generatedFiles...)

		files, err := listFiles(hostname)
		assert.NoError(t, err)
		assert.ElementsMatch(t, files, test_paths)
	})

	t.Run("every page listed in the sitemaps is crawled", func(t *testing.T) {
		var missing []string
		for i := range generatedSitemaps {
			for j := range generatedPagesPerSitemap {
				if p := generatedPagePath(i, j); !ts.wasVisited(p) {
					missing = append(missing, p)
				}
			}
		}
		for _, p := range []string{"/", "/1", "/2", "/3", "/500"} {
			if !ts.wasVisited(p) {
				missing = append(missing, p)
			}
		}
		assert.Empty(t, missing, "%d sitemap pages were never requested", len(missing))
	})

	t.Run("each file has been uploaded", func(t *testing.T) {
		files, err := listFiles(hostname)
		assert.NoError(t, err)

		assert.Equal(t, expectedFiles, uploader.UploadFileCallCount())

		var uploadedPaths []string
		for i := 0; i < uploader.UploadFileCallCount(); i++ {
			_, path, _, _ := uploader.UploadFileArgsForCall(i)
			uploadedPaths = append(uploadedPaths, path)
		}

		for _, testPath := range files {
			assert.Contains(t, uploadedPaths, testPath)
		}
	})

	t.Run("correct file uploaded counter metric", func(t *testing.T) {
		assert.Equal(t, float64(expectedFiles-1), testutil.ToFloat64(m.FileUploadCounter()))
	})

	t.Run("correct file upload failures counter metric", func(t *testing.T) {
		assert.Equal(t, float64(1), testutil.ToFloat64(m.FileUploadFailuresCounter()))
	})
}

func TestSortNewestFirst(t *testing.T) {
	entries := []entry{
		{key: "/1", val: "2025-11-05T11:00:00+00:00"},
		{key: "/3", val: defaultLastmod},
		{key: "/2", val: "2025-11-07T11:00:00+00:00"},
		{key: "/500", val: "2025-01-07T11:00:00+00:00"},
		{key: "/", val: "2025-11-06T11:00:00+00:00"},
	}

	sortNewestFirst(entries)

	var keys []string
	for _, e := range entries {
		keys = append(keys, e.key)
	}
	assert.Equal(t, []string{"/2", "/", "/1", "/500", "/3"}, keys)
}
