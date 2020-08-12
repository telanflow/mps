package mps

import (
	"net/http"
	"regexp"
	"strings"
)

// Filter is an request interceptor
type Filter interface {
	Match(req *http.Request) bool
}

// A wrapper that would convert a function to a Filter interface type
type FilterFunc func(req *http.Request) bool

// Filter.Match(req) <=> FilterFunc(req)
func (f FilterFunc) Match(req *http.Request) bool {
	return f(req)
}

// FilterHostMatches for request.Host
func FilterHostMatches(regexps ...*regexp.Regexp) Filter {
	return FilterFunc(func(req *http.Request) bool {
		for _, re := range regexps {
			if re.MatchString(req.Host) {
				return true
			}
		}
		return false
	})
}

// FilterHostIs returns a Filter, testing whether the host to which the request is directed to equal
// to one of the given strings
func FilterHostIs(hosts ...string) Filter {
	hostSet := make(map[string]bool)
	for _, h := range hosts {
		hostSet[h] = true
	}
	return FilterFunc(func(req *http.Request) bool {
		_, ok := hostSet[req.URL.Host]
		return ok
	})
}

// FilterUrlMatches returns a Filter testing whether the destination URL
// of the request matches the given regexp, with or without prefix
func FilterUrlMatches(re *regexp.Regexp) Filter {
	return FilterFunc(func(req *http.Request) bool {
		return re.MatchString(req.URL.Path) ||
			re.MatchString(req.URL.Host+req.URL.Path)
	})
}

// FilterUrlHasPrefix returns a Filter checking wether the destination URL the proxy client has requested
// has the given prefix, with or without the host.
// For example FilterUrlHasPrefix("host/x") will match requests of the form 'GET host/x', and will match
// requests to url 'http://host/x'
func FilterUrlHasPrefix(prefix string) Filter {
	return FilterFunc(func(req *http.Request) bool {
		return strings.HasPrefix(req.URL.Path, prefix) ||
			strings.HasPrefix(req.URL.Host+req.URL.Path, prefix) ||
			strings.HasPrefix(req.URL.Scheme+req.URL.Host+req.URL.Path, prefix)
	})
}

// FilterUrlIs returns a Filter, testing whether or not the request URL is one of the given strings
// with or without the host prefix.
// FilterUrlIs("google.com/","foo") will match requests 'GET /' to 'google.com', requests `'GET google.com/' to
// any host, and requests of the form 'GET foo'.
func FilterUrlIs(urls ...string) Filter {
	urlSet := make(map[string]bool)
	for _, u := range urls {
		urlSet[u] = true
	}
	return FilterFunc(func(req *http.Request) bool {
		_, pathOk := urlSet[req.URL.Path]
		_, hostAndOk := urlSet[req.URL.Host+req.URL.Path]
		return pathOk || hostAndOk
	})
}
