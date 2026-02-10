package page_comparer

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mirrorer/internal/page_fetcher"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

//counterfeiter:generate -o ./fakes/ . PageComparerInterface
type PageComparerInterface interface {
	HaveSameBody(pageA page_fetcher.Page, pageB page_fetcher.Page) (bool, error)
}

type PageComparer struct{}

// HaveSameBody takes two page_fetcher.Page structs, and compares their Body contents based on their ContentType.
// If ContentType is "text/html" it compares the text that would be visible to a user.
// If ContentType is not "text/html", the Body contents are compared as strings.
func (*PageComparer) HaveSameBody(pageA page_fetcher.Page, pageB page_fetcher.Page) (bool, error) {
	var mediaTypeA string
	if pageA.ContentType != "" {
		mediaType, _, err := mime.ParseMediaType(pageA.ContentType)
		if err != nil {
			return false, err
		}

		mediaTypeA = mediaType
	}

	var mediaTypeB string
	if pageB.ContentType != "" {
		mediaType, _, err := mime.ParseMediaType(pageB.ContentType)
		if err != nil {
			return false, err
		}
		mediaTypeB = mediaType
	}

	// Only do the mismatch check if both of them aren't empty
	if (mediaTypeA != "" && mediaTypeB != "") && mediaTypeA != mediaTypeB {
		return false, nil
	}

	switch mediaTypeA {
	case "text/html":
		return compareHtml(pageA, pageB)
	default:
		return compareStrings(pageA, pageB)
	}

}

func compareStrings(pageA page_fetcher.Page, pageB page_fetcher.Page) (bool, error) {
	return pageA.Body == pageB.Body, nil
}

func compareHtml(pageA page_fetcher.Page, pageB page_fetcher.Page) (bool, error) {
	docA, err := html.Parse(strings.NewReader(pageA.Body))
	if err != nil {
		return false, err
	}

	docB, err := html.Parse(strings.NewReader(pageB.Body))
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
// htmlPage is different.
func ExtractVisibleTextFromHTML(node *html.Node) string {
	var output strings.Builder
	var extractText func(*html.Node)

	// Recursive function to traverse the tree
	extractText = func(n *html.Node) {
		switch n.Type {
		case html.ElementNode:
			// Ignore head, style, link, script etc
			// Don't descend into them
			if slices.Contains([]string{"head", "meta", "style", "link", "script"}, n.Data) {
				return
			}

		// Text nodes contain text visible on the screen
		case html.TextNode:
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
