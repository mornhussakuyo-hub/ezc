package ui

import (
	"sort"
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

type menuMatch struct {
	label         string
	originalIndex int
	score         int
}

type searchableMenuItem struct {
	label         string
	originalIndex int
	terms         []string
}

func prepareMenuItems(items []string) []searchableMenuItem {
	prepared := make([]searchableMenuItem, len(items))
	for index, item := range items {
		fullPinyin, initials := pinyinSearchTerms(item)
		prepared[index] = searchableMenuItem{
			label:         item,
			originalIndex: index,
			terms:         []string{strings.ToLower(item), fullPinyin, initials},
		}
	}
	return prepared
}

func filterMenuItems(items []string, query string) []menuMatch {
	return filterPreparedMenuItems(prepareMenuItems(items), query)
}

func filterPreparedMenuItems(items []searchableMenuItem, query string) []menuMatch {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	matches := make([]menuMatch, 0, len(items))
	for _, item := range items {
		score, matched := menuItemScore(item.terms, normalizedQuery)
		if matched {
			matches = append(matches, menuMatch{label: item.label, originalIndex: item.originalIndex, score: score})
		}
	}
	if normalizedQuery != "" {
		sort.SliceStable(matches, func(left, right int) bool {
			return matches[left].score < matches[right].score
		})
	}
	return matches
}

func menuItemScore(terms []string, query string) (int, bool) {
	if query == "" {
		return 0, true
	}
	bestScore := 0
	matched := false
	for termIndex, term := range terms {
		score, ok := fuzzyScore(term, query)
		if !ok {
			continue
		}
		score += termIndex * 5
		if !matched || score < bestScore {
			bestScore = score
			matched = true
		}
	}
	return bestScore, matched
}

func pinyinSearchTerms(value string) (string, string) {
	fullArgs := pinyin.NewArgs()
	fullArgs.Fallback = preserveSearchRune
	initialArgs := fullArgs
	initialArgs.Style = pinyin.FirstLetter
	return strings.ToLower(strings.Join(pinyin.LazyPinyin(value, fullArgs), "")),
		strings.ToLower(strings.Join(pinyin.LazyPinyin(value, initialArgs), ""))
}

func preserveSearchRune(value rune, _ pinyin.Args) []string {
	return []string{string(unicode.ToLower(value))}
}

func fuzzyScore(candidate, query string) (int, bool) {
	if candidate == query {
		return 0, true
	}
	if strings.HasPrefix(candidate, query) {
		return 10 + len(candidate) - len(query), true
	}
	if index := strings.Index(candidate, query); index >= 0 {
		return 100 + index + len(candidate) - len(query), true
	}

	candidateRunes := []rune(candidate)
	queryRunes := []rune(query)
	queryIndex := 0
	firstMatch := -1
	lastMatch := -1
	for candidateIndex, value := range candidateRunes {
		if queryIndex >= len(queryRunes) || value != queryRunes[queryIndex] {
			continue
		}
		if firstMatch < 0 {
			firstMatch = candidateIndex
		}
		lastMatch = candidateIndex
		queryIndex++
	}
	if queryIndex != len(queryRunes) {
		return 0, false
	}
	gaps := lastMatch - firstMatch + 1 - len(queryRunes)
	return 1000 + gaps*10 + firstMatch + len(candidateRunes) - len(queryRunes), true
}
