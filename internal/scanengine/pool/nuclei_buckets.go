package pool

import (
	"sync"
	"time"
)

// NucleiTagBuckets holds one batch pool per nuclei template-set bucket.
type NucleiTagBuckets struct {
	mu           sync.Mutex
	pools        map[string]*Pool
	base         Config
	flushTimeout time.Duration
	onFlush      func(bucketKey string, ev FlushEvent)
}

// NewNucleiTagBuckets creates per-bucket nuclei URL pools.
func NewNucleiTagBuckets(dataDir string, batchSize int, flushTimeout time.Duration, onFlush func(bucketKey string, ev FlushEvent)) *NucleiTagBuckets {
	if batchSize < 1 {
		batchSize = 30
	}
	if flushTimeout <= 0 {
		flushTimeout = 10 * time.Second
	}
	return &NucleiTagBuckets{
		pools: make(map[string]*Pool),
		base: Config{
			BatchSize:    batchSize,
			FlushTimeout: flushTimeout,
			DataDir:      dataDir,
			FilePrefix:   "nuclei_batch",
			Label:        "nucleibucket",
		},
		flushTimeout: flushTimeout,
		onFlush:      onFlush,
	}
}

// Add enqueues a URL into the template-set bucket. bucketName is e.g. "jenkins" or "_fallback".
func (b *NucleiTagBuckets) Add(bucketName, url, assetID, lineageBucket string) {
	if bucketName == "" || url == "" {
		return
	}
	bucketKey := "nuclei:" + bucketName
	b.mu.Lock()
	p := b.pools[bucketName]
	if p == nil {
		name := bucketName
		cfg := b.base
		cfg.FilePrefix = "nuclei_batch_" + name
		cfg.Label = "nuclei:" + name
		p = New(cfg, func(ev FlushEvent) {
			if b.onFlush != nil {
				b.onFlush(bucketKey, ev)
			}
		})
		p.Start()
		b.pools[bucketName] = p
	}
	b.mu.Unlock()

	p.Add(Member{
		Value:     url,
		AssetID:   assetID,
		BucketKey: lineageBucket,
	})
}

// FlushAll forces all buckets to flush remaining members.
func (b *NucleiTagBuckets) FlushAll() {
	b.mu.Lock()
	pools := make([]*Pool, 0, len(b.pools))
	for _, p := range b.pools {
		pools = append(pools, p)
	}
	b.mu.Unlock()
	for _, p := range pools {
		p.FlushNow()
	}
}

// Stop stops all bucket pools and performs final flush.
func (b *NucleiTagBuckets) Stop() {
	b.mu.Lock()
	pools := make([]*Pool, 0, len(b.pools))
	for _, p := range b.pools {
		pools = append(pools, p)
	}
	b.pools = make(map[string]*Pool)
	b.mu.Unlock()
	for _, p := range pools {
		p.Stop()
	}
}

// Len returns total pending URLs across all buckets.
func (b *NucleiTagBuckets) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := 0
	for _, p := range b.pools {
		n += p.Len()
	}
	return n
}
