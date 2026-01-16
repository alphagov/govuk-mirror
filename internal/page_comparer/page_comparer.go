package page_comparer

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

var ErrNoBody = errors.New("no body tag found in HTML")

func HaveSameBody(pageA io.Reader, pageB io.Reader) (bool, error) {
	bodyA, err := extractHtmlBody(pageA)
	if err != nil {
		return false, err
	}

	bodyB, err := extractHtmlBody(pageB)
	if err != nil {
		return false, err
	}

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

func extractHtmlBody(page io.Reader) (string, error) {
	doc, err := html.Parse(page)
	if err != nil {
		return "", err
	}

	for node := range doc.Descendants() {
		if node.Type == html.ElementNode && node.Data == "body" {
			return renderNode(node)
		}
	}

	return "", ErrNoBody
}

func renderNode(node *html.Node) (string, error) {
	buf := strings.Builder{}
	err := html.Render(&buf, node)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func checksum(str string) (string, error) {
	shaSum := sha256.New()

	if _, err := io.Copy(shaSum, strings.NewReader(str)); err != nil {
		return "", fmt.Errorf("failed to write bytes to the hasher: %w", err)
	}

	return base64.StdEncoding.EncodeToString(shaSum.Sum(nil)), nil
}
