package page_comparer_test

import (
	"fmt"
	"mirrorer/internal/page_comparer"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func TestPageComparer_HaveSameBody(t *testing.T) {
	t.Run("can tolerate badly formed HTML", func(t *testing.T) {
		assert.True(t, true, `
			there are no tests for an HTML document that's badly formed
			because the HTML 5 parser does its very best to make something
			of anything given to it. We can basically assume it'll be able
			to parse whatever it's given'
		`)
	})

	t.Run("can tolerate an HTML document having no body tag", func(t *testing.T) {
		assert.True(t, true, `
			there are no tests for an HTML document that doesn't
			contain a body tag because the HTML parser will
			implicitly create nodes like body or head when they
			are needed to make a working document

			see the comments on the function html.Parse
		`)
	})

	t.Run("returns true if the two documents have the same body content", func(t *testing.T) {
		pageA := "<html><body><p>Hello</p></body></html>"
		pageB := "<html><body><p>Hello</p></body></html>"

		same, _ := page_comparer.HaveSameBody(pageA, pageB)
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

		same, _ := page_comparer.HaveSameBody(pageA, pageB)
		assert.True(t, same)
	})

	t.Run("returns false if the two documents have different text visible to the user", func(t *testing.T) {
		pageA := "<html><body><p>Hello</p></body></html>"
		pageB := "<html><body><p>Goodbye</p></body></html>"

		same, _ := page_comparer.HaveSameBody(pageA, pageB)
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

				same, _ := page_comparer.HaveSameBody(a, b)
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

	t.Run("if the first character of either string is '{' it treats the inputs as strings not markup", func(t *testing.T) {
		t.Run("and it considers two identical strings to be the same", func(t *testing.T) {
			pageA := `{"some": "json"}`
			pageB := `{"some": "json"}`

			same, _ := page_comparer.HaveSameBody(pageA, pageB)
			assert.True(t, same)
		})

		t.Run("and it considers two non-identical strings to not be the same", func(t *testing.T) {
			pageA := `{"some": "json"}`
			pageB := `["some", "json"]`

			same, _ := page_comparer.HaveSameBody(pageA, pageB)
			assert.False(t, same)
		})

		t.Run("and if only one input is HTML it treats them both as strings", func(t *testing.T) {
			pageA := `<html></html>`
			pageB := `["some", "json"]`

			same, _ := page_comparer.HaveSameBody(pageA, pageB)
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
			assert.Nil(t, err, "failed to parse input HTML: %s", tt.htmlInput)

			actual := page_comparer.ExtractVisibleTextFromHTML(node)
			assert.Equal(t, actual, tt.expected)
		})
	}
}
