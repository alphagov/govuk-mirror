package file

import (
	"fmt"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func RedirectHTMLBody(redirectURL string) []byte {
	body := fmt.Sprintf(`<!DOCTYPE html>
	<html lang="en">
	<head>
	<meta http-equiv="refresh" content="1; url=%[1]s">
	<title>Redirecting</title>
	</head>
	<body>
	<p>Redirecting you to <a href="%[1]s">%[1]s</a>.</p>
	</body>
	</html>`, redirectURL)

	return []byte(body)
}

func Save(u *url.URL, contentType string, body []byte) error {
	filePath, err := GenerateFilePath(u, contentType)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, body, 0644)
	if err != nil {
		return err
	}

	return nil
}

func GenerateFilePath(u *url.URL, contentType string) (string, error) {
	// Extract host and path from URL
	host := u.Hostname()
	path := u.EscapedPath()

	// Split the path into segments and remove any query strings
	segments := strings.Split(path, "?")[0]
	segmentsSlice := strings.Split(segments, "/")

	// If the last segment is empty, assign it to "index"
	if segmentsSlice[len(segmentsSlice)-1] == "" {
		segmentsSlice[len(segmentsSlice)-1] = "index"
	}

	lastSegment := segmentsSlice[len(segmentsSlice)-1]

	// Find the extension(s) based on the content type
	extensions, err := mime.ExtensionsByType(contentType)
	if err != nil {
		return "", err
	}

	if len(extensions) > 0 {
		// Check for existing extension in path
		existingExtenion := filepath.Ext(lastSegment)

		// Add extension if a valid extension exists for the content type and path doesn't already have a valid extension
		if !slices.Contains(extensions, existingExtenion) {
			segmentsSlice[len(segmentsSlice)-1] += extensions[len(extensions)-1]
		}
	} else {
		return "", fmt.Errorf("Error determining content type")
	}

	// Construct the final path by joining host and the rest of the segments
	finalPath := filepath.Join(append([]string{host}, segmentsSlice...)...)

	return finalPath, nil
}
