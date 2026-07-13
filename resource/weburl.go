package resource

import (
	"fmt"
	"net/url"

	tcurls "github.com/taskcluster/taskcluster-lib-urls"
)

// webUIPath builds a Taskcluster web UI URL for the given path, joining path
// segments already escaped by the caller (segments may legitimately contain
// a literal "/" of their own, e.g. a worker pool ID — see pathSegment).
func webUIPath(rootURL, path string) string {
	return tcurls.UI(rootURL, path)
}

// pathSegment percent-encodes id for use as a single URL path segment,
// matching the web UI's own encodeURIComponent calls — notably escaping a
// literal "/" (e.g. within a worker pool ID) to %2F rather than letting it
// introduce an extra path segment.
func pathSegment(id string) string {
	return url.PathEscape(id)
}

// withQuery appends a query parameter to path if value is non-empty.
func withQuery(path, key, value string) string {
	if value == "" {
		return path
	}
	return fmt.Sprintf("%s?%s=%s", path, key, url.QueryEscape(value))
}
