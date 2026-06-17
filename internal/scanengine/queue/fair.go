package queue

import "math"

type tierLoc struct {
	tier     int
	index    int
	inflight int
}

func countActiveBuckets(inflight map[string]int) int {
	n := 0
	for _, c := range inflight {
		if c > 0 {
			n++
		}
	}
	return n
}

func bucketEligible(bucketKey string, perBucketMax, activeBucketMax int, inflight map[string]int) bool {
	if bucketKey == "" {
		bucketKey = "default"
	}
	cur := inflight[bucketKey]
	if cur >= perBucketMax {
		return false
	}
	if cur == 0 && countActiveBuckets(inflight) >= activeBucketMax {
		return false
	}
	return true
}

func bucketInflightCount(bucketKey string, inflight map[string]int) int {
	if bucketKey == "" {
		bucketKey = "default"
	}
	return inflight[bucketKey]
}

func (q *PriorityQueue) popAtTier(tier, index int) (Item, bool) {
	var item Item
	switch tier {
	case 0:
		item = q.high[index]
		q.high = append(q.high[:index], q.high[index+1:]...)
	case 1:
		item = q.med[index]
		q.med = append(q.med[:index], q.med[index+1:]...)
	case 2:
		item = q.low[index]
		q.low = append(q.low[:index], q.low[index+1:]...)
	default:
		return Item{}, false
	}
	delete(q.seen, item.WorkID)
	return item, true
}

func considerFairItem(
	tier, index int,
	item Item,
	inflight map[string]int,
	best *tierLoc,
	bestPri *Priority,
) {
	inflightCount := bucketInflightCount(item.BucketKey, inflight)

	// Lower inflight count wins; then priority tier; then priority value; then FIFO.
	if best.index >= 0 {
		if inflightCount > best.inflight {
			return
		}
		if inflightCount == best.inflight {
			if tier > best.tier {
				return
			}
			if tier == best.tier && item.Priority > *bestPri {
				return
			}
			if tier == best.tier && item.Priority == *bestPri && index >= best.index {
				return
			}
		}
	}

	*best = tierLoc{tier: tier, index: index, inflight: inflightCount}
	*bestPri = item.Priority
}

func (q *PriorityQueue) popFairFromItems(
	items []Item,
	tier int,
	minStage *StageRank,
	perBucketMax, activeBucketMax int,
	inflight map[string]int,
	best *tierLoc,
	bestPri *Priority,
) {
	for i, item := range items {
		if minStage != nil && ActionToStageRank(item.Action) != *minStage {
			continue
		}
		if !bucketEligible(item.BucketKey, perBucketMax, activeBucketMax, inflight) {
			continue
		}
		considerFairItem(tier, i, item, inflight, best, bestPri)
	}
}

// PopFair removes the highest-priority eligible item while spreading load across buckets.
func (q *PriorityQueue) PopFair(perBucketMax, activeBucketMax int, inflight map[string]int) (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	best := tierLoc{tier: 3, index: -1, inflight: math.MaxInt32}
	var bestPri Priority = PriorityLow + 1

	q.popFairFromItems(q.high, 0, nil, perBucketMax, activeBucketMax, inflight, &best, &bestPri)
	q.popFairFromItems(q.med, 1, nil, perBucketMax, activeBucketMax, inflight, &best, &bestPri)
	q.popFairFromItems(q.low, 2, nil, perBucketMax, activeBucketMax, inflight, &best, &bestPri)

	if best.index < 0 {
		return Item{}, false
	}
	return q.popAtTier(best.tier, best.index)
}

func minStageInQueue(q *PriorityQueue) (StageRank, bool) {
	minRank := StageRank(math.MaxInt)
	found := false
	scan := func(items []Item) {
		for _, item := range items {
			r := ActionToStageRank(item.Action)
			if r < minRank {
				minRank = r
				found = true
			}
		}
	}
	scan(q.high)
	scan(q.med)
	scan(q.low)
	if !found {
		return 0, false
	}
	return minRank, true
}

// PopFairStaged pops from the earliest incomplete stage, with fair bucket scheduling within that stage.
func (q *PriorityQueue) PopFairStaged(perBucketMax, activeBucketMax int, inflight map[string]int) (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	minStage, ok := minStageInQueue(q)
	if !ok {
		return Item{}, false
	}

	best := tierLoc{tier: 3, index: -1, inflight: math.MaxInt32}
	var bestPri Priority = PriorityLow + 1

	q.popFairFromItems(q.high, 0, &minStage, perBucketMax, activeBucketMax, inflight, &best, &bestPri)
	q.popFairFromItems(q.med, 1, &minStage, perBucketMax, activeBucketMax, inflight, &best, &bestPri)
	q.popFairFromItems(q.low, 2, &minStage, perBucketMax, activeBucketMax, inflight, &best, &bestPri)

	if best.index < 0 {
		return Item{}, false
	}
	return q.popAtTier(best.tier, best.index)
}

// StageDepth returns pending queue depth grouped by stage rank.
func (q *PriorityQueue) StageDepth() map[StageRank]int {
	q.mu.Lock()
	defer q.mu.Unlock()

	depth := make(map[StageRank]int)
	add := func(items []Item) {
		for _, item := range items {
			depth[ActionToStageRank(item.Action)]++
		}
	}
	add(q.high)
	add(q.med)
	add(q.low)
	return depth
}
