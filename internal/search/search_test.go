package search

import "testing"

func TestMatcherSupportsFullPinyinAndInitials(t *testing.T) {
	items := []string{"草稿.txt", "测试报告.txt", "测试版本.txt"}
	matcher := New(items)

	fullPinyin, ok := matcher.Best("ceshibaogao")
	if !ok || fullPinyin.Index != 1 || fullPinyin.Kind != Exact {
		t.Fatalf("unexpected full pinyin result: %#v, %v", fullPinyin, ok)
	}

	initials, ok := matcher.Best("csbg")
	if !ok || initials.Index != 1 || initials.Kind != Exact {
		t.Fatalf("unexpected initials result: %#v, %v", initials, ok)
	}
}

func TestMatcherRanksExactBeforeSubstringBeforeSubsequence(t *testing.T) {
	items := []string{"a-b-c.txt", "zabc.txt", "abc"}
	results := New(items).Rank("abc")
	if len(results) != 3 {
		t.Fatalf("expected three matches, got %#v", results)
	}
	expected := []Result{{Index: 2, Kind: Exact}, {Index: 1, Kind: Substring}, {Index: 0, Kind: Subsequence}}
	for index := range expected {
		if results[index] != expected[index] {
			t.Fatalf("expected %#v, got %#v", expected, results)
		}
	}
}

func TestMatcherPrefersExactStemPinyin(t *testing.T) {
	items := []string{"前缀测试报告.txt", "测试报告.txt"}
	result, ok := New(items).Best("ceshibaogao")
	if !ok || result.Index != 1 || result.Kind != Exact {
		t.Fatalf("expected exact stem pinyin match, got %#v, %v", result, ok)
	}
}

func TestMatcherPreservesOriginalOrderWithinSameKind(t *testing.T) {
	items := []string{"second-abc.txt", "first-abc.txt", "third-abc.txt"}
	results := New(items).Rank("abc")
	for index, result := range results {
		if result.Index != index || result.Kind != Substring {
			t.Fatalf("expected stable original order, got %#v", results)
		}
	}
}

func TestMatcherUsesPinyinSubsequence(t *testing.T) {
	result, ok := New([]string{"测试报告.txt", "测试版本.txt"}).Best("cshg")
	if !ok || result.Index != 0 || result.Kind != Subsequence {
		t.Fatalf("unexpected subsequence result: %#v, %v", result, ok)
	}
}

func TestMatcherReturnsNoResult(t *testing.T) {
	if result, ok := New([]string{"测试报告.txt"}).Best("xyz"); ok {
		t.Fatalf("expected no result, got %#v", result)
	}
}
