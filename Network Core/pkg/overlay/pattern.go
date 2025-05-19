package overlay

import (
    "regexp"
    "strings"
)

// patternToRegex converts a URL pattern to a regular expression
func patternToRegex(pattern string) *regexp.Regexp {
    // Escape special regex characters
    pattern = regexp.QuoteMeta(pattern)

    // Replace path parameters with regex groups
    pattern = strings.ReplaceAll(pattern, "/\\{[^}]+\\}", "/([^/]+)")
    pattern = "^" + pattern + "$"

    return regexp.MustCompile(pattern)
}
