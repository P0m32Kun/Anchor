package seed

// JunkKeywordRule describes a passive-search junk keyword rule.
type JunkKeywordRule struct {
	Keyword      string
	WordBoundary bool
}

var runtimeJunkKeywords []JunkKeywordRule

func init() {
	runtimeJunkKeywords = append([]JunkKeywordRule{}, defaultJunkKeywords...)
}

// SetJunkKeywords replaces the active built-in junk keyword list (from scan.config.yaml).
func SetJunkKeywords(rules []JunkKeywordRule) {
	if len(rules) == 0 {
		return
	}
	runtimeJunkKeywords = append([]JunkKeywordRule{}, rules...)
}

func activeJunkKeywords(extra []junkKeywordRule) []junkKeywordRule {
	out := make([]junkKeywordRule, 0, len(runtimeJunkKeywords)+len(extra))
	for _, r := range runtimeJunkKeywords {
		out = append(out, junkKeywordRule(r))
	}
	out = append(out, extra...)
	return out
}

// defaultJunkKeywords is the compiled fallback when scan.config.yaml is absent.
var defaultJunkKeywords = []JunkKeywordRule{
	// Chinese — gambling / lottery
	{Keyword: "博彩"},
	{Keyword: "赌博"},
	{Keyword: "彩票"},
	{Keyword: "娱乐城"},
	{Keyword: "赌场"},
	{Keyword: "六合彩"},
	{Keyword: "时时彩"},
	{Keyword: "百家乐"},
	{Keyword: "棋牌"},
	{Keyword: "网赌"},
	// Chinese — ads / spam / illegal
	{Keyword: "色情"},
	{Keyword: "代发"},
	{Keyword: "网赚"},
	{Keyword: "刷单"},
	{Keyword: "私服"},
	// English — gambling
	{Keyword: "casino", WordBoundary: true},
	{Keyword: "gambling", WordBoundary: true},
	{Keyword: "porn", WordBoundary: true},
	{Keyword: "xxx", WordBoundary: true},
	{Keyword: "lottery", WordBoundary: true},
}

type junkKeywordRule struct {
	Keyword      string
	WordBoundary bool
}
