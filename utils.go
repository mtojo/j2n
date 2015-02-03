package main

import (
	"path/filepath"
	"regexp"
	"strings"
)

var upperCamelCaseRegex = regexp.MustCompile("[A-Z]*[0-9a-z]*")

func inSlice(slice []string, s string) bool {
	for _, elem := range slice {
		if elem == s {
			return true
		}
	}
	return false
}

func appendUnique(slice []string, s string) []string {
  if inSlice(slice, s) {
    return slice
  }
	return append(slice, s)
}

func splitString(s, delim string) []string {
	if s == "" {
		return []string{}
	} else {
		return strings.Split(s, delim)
	}
}

func filter(s []string, fn func(string) bool) []string {
	var c []string
	for _, elem := range s {
		if fn(elem) {
			c = append(c, elem)
		}
	}
	return c
}

func snakeCase(s string) string {
	chunks := upperCamelCaseRegex.FindAllString(s, -1)
	for idx, val := range chunks {
		chunks[idx] = strings.ToLower(val)
	}
	return strings.Join(chunks, "_")
}

func upperCamelCase(s string) string {
	chunks := upperCamelCaseRegex.FindAllString(s, -1)
	for idx, val := range chunks {
		chunks[idx] = strings.Title(val)
	}
	return strings.Join(chunks, "")
}

func replaceExt(fname, ext string) string {
	return strings.TrimSuffix(fname, filepath.Ext(fname)) + "." + ext
}
