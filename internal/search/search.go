package search

import (
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

type Kind int

const (
	Exact Kind = iota
	Substring
	Subsequence
)

type Result struct {
	Index int
	Kind  Kind
}

type Matcher struct {
	items []searchableItem
}

type searchableItem struct {
	index int
	terms []string
}

func New(items []string) *Matcher {
	prepared := make([]searchableItem, len(items))
	for index, item := range items {
		fullPinyin, initials := pinyinTerms(item)
		stem := strings.TrimSuffix(item, filepath.Ext(item))
		stemFullPinyin, stemInitials := pinyinTerms(stem)
		prepared[index] = searchableItem{
			index: index,
			terms: []string{
				strings.ToLower(item),
				strings.ToLower(stem),
				fullPinyin,
				stemFullPinyin,
				initials,
				stemInitials,
			},
		}
	}
	return &Matcher{items: prepared}
}

func (matcher *Matcher) Rank(query string) []Result {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	results := make([]Result, 0, len(matcher.items))
	for _, item := range matcher.items {
		kind, matched := itemMatchKind(item.terms, normalizedQuery)
		if matched {
			results = append(results, Result{Index: item.index, Kind: kind})
		}
	}
	if normalizedQuery != "" {
		sort.SliceStable(results, func(left, right int) bool {
			return results[left].Kind < results[right].Kind
		})
	}
	return results
}

func (matcher *Matcher) Best(query string) (Result, bool) {
	results := matcher.Rank(query)
	if len(results) == 0 {
		return Result{}, false
	}
	return results[0], true
}

func itemMatchKind(terms []string, query string) (Kind, bool) {
	if query == "" {
		return Exact, true
	}
	bestKind := Subsequence
	matched := false
	for _, term := range terms {
		kind, ok := matchKind(term, query)
		if !ok {
			continue
		}
		if !matched || kind < bestKind {
			bestKind = kind
			matched = true
		}
	}
	return bestKind, matched
}

func matchKind(candidate, query string) (Kind, bool) {
	if candidate == query {
		return Exact, true
	}
	if strings.Contains(candidate, query) {
		return Substring, true
	}
	if isSubsequence(candidate, query) {
		return Subsequence, true
	}
	return Subsequence, false
}

func isSubsequence(candidate, query string) bool {
	queryRunes := []rune(query)
	queryIndex := 0
	for _, value := range candidate {
		if queryIndex < len(queryRunes) && value == queryRunes[queryIndex] {
			queryIndex++
		}
	}
	return queryIndex == len(queryRunes)
}

func pinyinTerms(value string) (string, string) {
	fullArgs := pinyin.NewArgs()
	fullArgs.Fallback = preserveRune
	initialArgs := fullArgs
	initialArgs.Style = pinyin.FirstLetter
	return strings.ToLower(strings.Join(pinyin.LazyPinyin(value, fullArgs), "")),
		strings.ToLower(strings.Join(pinyin.LazyPinyin(value, initialArgs), ""))
}

func preserveRune(value rune, _ pinyin.Args) []string {
	return []string{string(unicode.ToLower(value))}
}
