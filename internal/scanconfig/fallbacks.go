package scanconfig

// fallbackJunkKeywords mirrors internal/scanengine/seed/junk_keywords.go when no file is present.
func fallbackJunkKeywords() []JunkKeyword {
	return []JunkKeyword{
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
		{Keyword: "色情"},
		{Keyword: "代发"},
		{Keyword: "网赚"},
		{Keyword: "刷单"},
		{Keyword: "私服"},
		{Keyword: "casino", WordBoundary: true},
		{Keyword: "gambling", WordBoundary: true},
		{Keyword: "porn", WordBoundary: true},
		{Keyword: "xxx", WordBoundary: true},
		{Keyword: "lottery", WordBoundary: true},
	}
}
