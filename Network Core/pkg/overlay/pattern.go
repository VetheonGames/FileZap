package overlay

import "strings"

// isPatternMatch checks if a path matches a pattern
func isPatternMatch(pattern, path string) bool {
    patternParts := strings.Split(pattern, "/")
    pathParts := strings.Split(path, "/")

    if len(patternParts) != len(pathParts) {
        return false
    }

    for i := 0; i < len(patternParts); i++ {
        if patternParts[i] == "" && pathParts[i] == "" {
            continue
        }
        if patternParts[i] == "" || pathParts[i] == "" {
            return false
        }
        if !isPartMatch(patternParts[i], pathParts[i]) {
            return false
        }
    }

    return true
}

// isPartMatch checks if a path part matches a pattern part
func isPartMatch(pattern, part string) bool {
    if strings.HasPrefix(pattern, "{") && strings.HasSuffix(pattern, "}") {
        return true
    }
    return pattern == part
}
