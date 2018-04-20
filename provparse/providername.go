package provparse

import (
	"regexp"
)

// providerNameRegexp is a regular expression that can match the opinionated
// repository name syntax for the registry.
var providerNameRegexp = regexp.MustCompile(
	`^` + // Beginning
		`terraform-provider-` + // Always starts with `terraform-provider-`
		`(?P<name>[a-z0-9]|[a-z0-9][-_.a-z0-9]*[a-z0-9])` + // Required name
		`$`)

// providerName extracts the provider name from a repository name. The boolean
// result is false if the input format didn't match the opinionated format.
func providerName(input string) (name string, ok bool) {
	// Regexp match
	match := providerNameRegexp.FindStringSubmatch(input)
	if match == nil {
		ok = false
		return
	}

	// Get all the named captures
	captures := make(map[string]string)
	for i, name := range providerNameRegexp.SubexpNames() {
		if i != 0 {
			captures[name] = match[i]
		}
	}

	// Set the results
	name = captures["name"]
	ok = true
	return
}
