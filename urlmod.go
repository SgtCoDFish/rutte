package rutte

import (
	"net/url"
	"strings"
)

// ModURL converts a relative URL to a nextjs friendly format. This involves:
// - adding .md for github-browsing friendliness
// - removing one level of "..", if any exist
// pathChecker should return true if the given filename exists, and false otherwise;
// this makes it possible to check whether we need to add "README.md" to the path
func ModURL(urlIn string, pathChecker func(string) bool) (string, error) {
	parsed, err := url.Parse(urlIn)
	if err != nil {
		return "", err
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")

	if strings.HasPrefix(parsed.Path, "../../") {
		parsed.Path = strings.TrimPrefix(parsed.Path, "../")
	}

	fileExists := pathChecker(parsed.Path)
	if fileExists {
		parsed.Path += ".md"
	} else {
		parsed.Path += "/README.md"
	}

	return parsed.String(), nil
}
