package main

import (
	"regexp"
	"sort"
	"strings"

	"github.com/antzucaro/matchr"
)

var reg = regexp.MustCompile("[^a-z]+")

var uselessWords = []string{
	"le", "la", "et",
	"de", "het", "en",
	"the", "and", "a", "an",
}

func generalizer(s string) string {
	s = " " + s + " "
	s = reg.ReplaceAllString(strings.ToLower(s), " ")
	for _, w := range uselessWords {
		s = strings.Replace(s, " "+w+" ", " ", -1)
	}
	keys := getMetaphoneKeys(s)
	s = strings.Join(keys, "")

	return s
}

func getLowercasedSlice(s string) []string {
	var returnParts []string
	parts := strings.Split(s, " ")
	for _, part := range parts {
		cleaned := reg.ReplaceAllString(strings.ToLower(part), "")
		returnParts = append(returnParts, cleaned)
	}

	returnParts = unique(returnParts)
	sort.Strings(returnParts)

	return returnParts
}

func getMetaphoneKeys(s string) []string {
	parts := metaphonify(s)

	parts = unique(parts)

	sort.Strings(parts)

	return parts
}

func metaphonify(s string) []string {
	var nameParts []string
	names := strings.Split(s, " ")
	for _, name := range names {
		cleaned := reg.ReplaceAllString(strings.ToLower(name), "")
		a, _ := matchr.DoubleMetaphone(cleaned)
		if len(a) >= 1 {
			nameParts = append(nameParts, a)
		}
	}
	sort.Strings(nameParts)
	return nameParts
}

func unique(input []string) []string {
	u := make([]string, 0, len(input))
	m := make(map[string]bool)

	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}

	return u
}
