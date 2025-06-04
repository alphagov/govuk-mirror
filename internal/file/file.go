package file

import (
	"fmt"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var cssUrlRegex = regexp.MustCompile(`url\(["']?(.*?)["']?\)`)

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

	var extensions []string
	var err error

	if contentType != "" {
		// Find the extension(s) based on the content type
		extensions, err = mime.ExtensionsByType(contentType)
		if err != nil {
			return "", err
		}
	}

	existingExtenion := filepath.Ext(lastSegment)

	if len(extensions) == 0 && existingExtenion == "" {
		return "", fmt.Errorf("error determining content type")
	}

	if len(extensions) > 0 && !slices.Contains(extensions, existingExtenion) {
		segmentsSlice[len(segmentsSlice)-1] += extensions[len(extensions)-1]
	}

	// Construct the final path by joining host and the rest of the segments
	finalPath := filepath.Join(append([]string{host}, segmentsSlice...)...)

	return finalPath, nil
}

func FindCssUrls(body []byte) []string {
	urls := cssUrlRegex.FindAllStringSubmatch(string(body), -1)
	result := []string{}
	for _, url := range urls {
		result = append(result, url[1])
	}
	return result
}
