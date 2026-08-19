package api

import (
	"fmt"
	"strconv"
	"strings"
)

// parseVersion splits a dot-separated version string such as "15.0.1" into its
// numeric components. An optional leading "v" is ignored, and parsing stops at
// the first component that does not begin with a digit, so suffixed versions
// like "15.0-beta" parse as [15, 0].
func parseVersion(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if len(s) > 0 && (s[0] == 'v' || s[0] == 'V') {
		s = s[1:]
	}

	var parts []int
	for _, field := range strings.Split(s, ".") {
		end := 0
		for end < len(field) && field[end] >= '0' && field[end] <= '9' {
			end++
		}
		if end == 0 {
			break
		}
		n, err := strconv.Atoi(field[:end])
		if err != nil {
			break
		}
		parts = append(parts, n)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("unrecognized version string %q", s)
	}
	return parts, nil
}

// CompareVersions returns -1 if a < b, 0 if they are equal, and 1 if a > b.
// Missing trailing components count as zero, so "15.0" equals "15.0.0".
func CompareVersions(a, b string) (int, error) {
	av, err := parseVersion(a)
	if err != nil {
		return 0, err
	}
	bv, err := parseVersion(b)
	if err != nil {
		return 0, err
	}
	for i := 0; i < len(av) || i < len(bv); i++ {
		var x, y int
		if i < len(av) {
			x = av[i]
		}
		if i < len(bv) {
			y = bv[i]
		}
		if x != y {
			if x < y {
				return -1, nil
			}
			return 1, nil
		}
	}
	return 0, nil
}

// ServerAtLeast reports whether the connected server's version is at least
// min. It also returns the version the server reported, for error messages.
func (c *Client) ServerAtLeast(min string) (bool, string, error) {
	version, err := c.ServerVersion()
	if err != nil {
		return false, "", err
	}
	cmp, err := CompareVersions(version, min)
	if err != nil {
		return false, version, err
	}
	return cmp >= 0, version, nil
}
