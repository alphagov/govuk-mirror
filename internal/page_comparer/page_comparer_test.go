package page_comparer_test

import (
	"fmt"
	"mirrorer/internal/page_comparer"
	"mirrorer/internal/page_fetcher"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func htmlPage(body string) page_fetcher.Page {
	return page_fetcher.Page{
		Body:        body,
		ContentType: "text/html",
	}
}

func jsonPage(body string) page_fetcher.Page {
	return page_fetcher.Page{
		Body:        body,
		ContentType: "application/json; charset=utf-8",
	}
}

func TestPageComparer_HaveSameBody(t *testing.T) {
	comparer := page_comparer.PageComparer{}

	t.Run("when the content type is text/html", func(t *testing.T) {
		t.Run("can tolerate badly formed HTML", func(t *testing.T) {
			// the HTML 5 parser does its very best to make something
			// of anything given to it. We can basically assume it'll be able
			// to parse whatever it's given. This test is a very simple
			// demonstration of that.
			body := "<html <body"
			same, err := comparer.HaveSameBody(htmlPage(body), htmlPage(body))
			assert.NoError(t, err)
			assert.True(t, same)
		})

		t.Run("can tolerate an HTML document having no body tag", func(t *testing.T) {
			// the HTML parser will
			// implicitly create nodes like body or head when they
			// are needed to make a working document
			//
			// see the comments on the function html.Parse
			body := "<html><head><title>Hello!</title></head></html>"
			same, err := comparer.HaveSameBody(htmlPage(body), htmlPage(body))
			assert.NoError(t, err)
			assert.True(t, same)
		})

		t.Run("returns true if the two documents have the same body content", func(t *testing.T) {
			pageA := "<html><body><p>Hello</p></body></html>"
			pageB := "<html><body><p>Hello</p></body></html>"

			same, _ := comparer.HaveSameBody(htmlPage(pageA), htmlPage(pageB))
			assert.True(t, same)
		})

		t.Run("returns true if the two documents have the same text visible to the user, but other elements different", func(t *testing.T) {
			pageA := `<html><body>
			<p>Hello</p>
			<script>alert("Script");</script>
		</body></html>`
			pageB := `<html><body>
			<p>Hello</p>
			<link rel="stylesheet" src="style.css" />
		</body></html>`

			same, _ := comparer.HaveSameBody(htmlPage(pageA), htmlPage(pageB))
			assert.True(t, same)
		})

		t.Run("returns false if the two documents have different text visible to the user", func(t *testing.T) {
			pageA := "<html><body><p>Hello</p></body></html>"
			pageB := "<html><body><p>Goodbye</p></body></html>"

			same, _ := comparer.HaveSameBody(htmlPage(pageA), htmlPage(pageB))
			assert.False(t, same)
		})

		t.Run("always returns true for two documents which would have the same body content when parsed", func(t *testing.T) {
			testCases := [][]string{
				{"hello", "<body>hello</body>"},
				{"<p>hello</p>", "<body><p>hello</p></body>"},
				{"", "<html><head><title>empty</title></head></html>"},
			}

			for _, testCase := range testCases {
				t.Run(fmt.Sprintf("'%s' and '%s'", testCase[0], testCase[1]), func(t *testing.T) {
					a := testCase[0]
					b := testCase[1]

					same, _ := comparer.HaveSameBody(htmlPage(a), htmlPage(b))
					assert.True(
						t,
						same,
						"expected two documents to end up with the same body contents; '%s' and '%s'",
						testCase[0],
						testCase[1],
					)
				})
			}
		})

		t.Run("will correctly parse content type containing a charset", func(t *testing.T) {
			body := "<html><body><p>Hello!</p></body></html>"
			pageA := page_fetcher.Page{
				Body:        body,
				ContentType: "text/html; charset=utf-8",
			}
			pageB := page_fetcher.Page{
				Body:        body,
				ContentType: "text/html; charset=utf-8",
			}
			same, err := comparer.HaveSameBody(pageA, pageB)
			assert.NoError(t, err)
			assert.True(t, same)
		})
	})

	t.Run("when the content type is", func(t *testing.T) {
		// the content types deliberately have a mix of
		// charsets and no charsets to ensure we handle both cases
		contentTypes := []string{
			"text/css",
			"application/javascript",
			"application/xml; charset=utf-8",
			"application/rss+xml",
			"application/octet-stream",
			"text/text; charset=utf-8",
			"application/vnd.ms-excel",
		}

		for _, contentType := range contentTypes {
			t.Run(contentType, func(t *testing.T) {
				t.Run("it considers two identical strings to be the same", func(t *testing.T) {
					pageA := page_fetcher.Page{
						Body:        "same",
						ContentType: contentType,
					}
					pageB := page_fetcher.Page{
						Body:        "same",
						ContentType: contentType,
					}

					same, _ := comparer.HaveSameBody(pageA, pageB)
					assert.True(t, same)
				})

				t.Run("it considers two non-identical strings to not be the same", func(t *testing.T) {
					pageA := page_fetcher.Page{
						Body:        "same",
						ContentType: contentType,
					}
					pageB := page_fetcher.Page{
						Body:        "different",
						ContentType: contentType,
					}

					same, _ := comparer.HaveSameBody(pageA, pageB)
					assert.False(t, same)
				})
			})
		}

	})

	t.Run("when the content types are different", func(t *testing.T) {
		t.Run("will always return false", func(t *testing.T) {
			same, _ := comparer.HaveSameBody(jsonPage("same"), htmlPage("same"))
			assert.False(t, same)

			same, _ = comparer.HaveSameBody(jsonPage("same"), htmlPage("different"))
			assert.False(t, same)
		})
	})

	t.Run("when either content type is missing", func(t *testing.T) {
		t.Run("will compare the pages if only page a's content type is missing", func(t *testing.T) {
			pageA := page_fetcher.Page{
				Body:        "Body",
				ContentType: "",
			}
			pageB := page_fetcher.Page{
				Body:        "Body",
				ContentType: "text/html; charset=utf-8",
			}

			same, err := comparer.HaveSameBody(pageA, pageB)
			assert.NoError(t, err)
			assert.True(t, same)
		})

		t.Run("will compare the pages if only page b's content type is missing", func(t *testing.T) {
			pageA := page_fetcher.Page{
				Body:        "Body",
				ContentType: "text/html; charset=utf-8",
			}
			pageB := page_fetcher.Page{
				Body:        "Body",
				ContentType: "",
			}

			same, err := comparer.HaveSameBody(pageA, pageB)
			assert.NoError(t, err)
			assert.True(t, same)
		})

		t.Run("will compare them as strings", func(t *testing.T) {
			pageA := page_fetcher.Page{
				Body:        "<p>Body</p>",
				ContentType: "",
			}
			pageB := page_fetcher.Page{
				Body:        "<span>Body</span>",
				ContentType: "text/html; charset=utf-8",
			}

			same, err := comparer.HaveSameBody(pageA, pageB)
			assert.NoError(t, err)
			assert.False(t, same)
		})
	})
}

func TestExtractVisibleTextFromHTML(t *testing.T) {
	tests := []struct {
		name      string
		htmlInput string
		expected  string
	}{
		{
			name:      "Ignored tag only",
			htmlInput: "<script>alert('Hello!');</script>",
			expected:  "",
		},
		{
			name:      "Empty body",
			htmlInput: "<html><body></body></html>",
			expected:  "",
		},
		{
			name:      "Empty string",
			htmlInput: "",
			expected:  "",
		},
		{
			name:      "Document with one visible text",
			htmlInput: "<html><body><h1>Hello, World!</h1></body></html>",
			expected:  "Hello, World!",
		},
		{
			name: "Long document with many pieces of visible text",
			htmlInput: `
				<!DOCTYPE html>
				<html>
				<head>
					<title>Ignored</title>
					<style>body { background: #f00; }</style>
				</head>
				<body>
					<h1>Header</h1>
					<p>
						Paragraph one containing line
						breaks and indenting spaces
					</p>
					<p>Paragraph two.</p>
					<div>
						<span>Text inside a span.</span>
					</div>
					<script>console.log("Ignored script");</script>
				</body>
				</html>
			`,
			expected: "Header\nParagraph one containing line\n\t\t\t\t\t\tbreaks and indenting spaces\nParagraph two.\nText inside a span.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := html.Parse(strings.NewReader(tt.htmlInput))
			assert.NoError(t, err, "failed to parse input HTML: %s", tt.htmlInput)

			actual := page_comparer.ExtractVisibleTextFromHTML(node)
			assert.Equal(t, actual, tt.expected)
		})
	}
}
