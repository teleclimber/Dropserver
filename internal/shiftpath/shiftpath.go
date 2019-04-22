package shiftpath

import (
	"path"
	"strings"
)

// ShiftPath takes a path and returns the first element and the remainder
func ShiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}
	return p[1:i], p[i:]
}

// TODO: really need to consider what happens in case of query string parameters /foo/bar?baz
