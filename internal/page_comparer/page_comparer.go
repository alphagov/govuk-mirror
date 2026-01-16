package page_comparer

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// HaveSameBody takes two strings, which it assumes to be HTML, and compares the text in that page which
// would be visible to a user. If the text is the same, the page bodies are considered to be the same.
func HaveSameBody(pageA io.Reader, pageB io.Reader) (bool, error) {
	docA, err := html.Parse(pageA)
	if err != nil {
		return false, err
	}

	docB, err := html.Parse(pageB)
	if err != nil {
		return false, err
	}

	bodyA := ExtractVisibleTextFromHTML(docA)
	bodyB := ExtractVisibleTextFromHTML(docB)

	checksumA, err := checksum(bodyA)
	if err != nil {
		return false, err
	}

	checksumB, err := checksum(bodyB)
	if err != nil {
		return false, err
	}

	return checksumA == checksumB, nil
}

// ExtractVisibleTextFromHTML takes an *html.Node and recurses its descendents
// to find any and all text that would be visible to a user in their browser.
//
// We need this because there is lots of text in HTML that can be different
// between two documents without affecting whether it would appear different
// to a user. For our purposes, we only care about whether the text on the
// page is different.
func ExtractVisibleTextFromHTML(node *html.Node) string {
	var output strings.Builder
	var extractText func(*html.Node)

	// Recursive function to traverse the tree
	extractText = func(n *html.Node) {
		// Ignore head, style, link, script etc
		// Don't descend into them
		if n.Type == html.ElementNode {
			switch n.Data {
			case "head", "meta", "style", "link", "script":
				return
			}
		}

		// Text nodes contain text visible on the screen
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				output.WriteString(text + "\n")
			}
		}

		// Recursively process child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}

	// Start the recursion from the root node
	extractText(node)

	return strings.TrimSpace(output.String())
}

func checksum(str string) (string, error) {
	shaSum := sha256.New()

	if _, err := io.Copy(shaSum, strings.NewReader(str)); err != nil {
		return "", fmt.Errorf("failed to write bytes to the hasher: %w", err)
	}

	return base64.StdEncoding.EncodeToString(shaSum.Sum(nil)), nil
}
