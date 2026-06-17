package pool

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// IPPortAggregator merges open ports per IP into one nmap batch work per IP.
type IPPortAggregator struct {
	mu      sync.Mutex
	byIP    map[string]*ipPortEntry
	dataDir string
	gen     int
	onFlush func(ip string, ev FlushEvent)
}

type ipPortEntry struct {
	ip      string
	ports   map[int]struct{}
	members []Member
}

// NewIPPortAggregator creates an aggregator. onFlush is called once per IP on FlushAll.
func NewIPPortAggregator(dataDir string, onFlush func(ip string, ev FlushEvent)) *IPPortAggregator {
	return &IPPortAggregator{
		byIP:    make(map[string]*ipPortEntry),
		dataDir: dataDir,
		onFlush: onFlush,
	}
}

// Add records an open port for an IP. Duplicate ip:port pairs are ignored.
func (a *IPPortAggregator) Add(ip string, port int, assetID, bucketKey string) {
	ip = strings.TrimSpace(ip)
	if ip == "" || port <= 0 {
		return
	}
	key := strings.ToLower(ip)
	value := fmt.Sprintf("%s:%d", ip, port)

	a.mu.Lock()
	defer a.mu.Unlock()

	entry := a.byIP[key]
	if entry == nil {
		entry = &ipPortEntry{
			ip:    ip,
			ports: make(map[int]struct{}),
		}
		a.byIP[key] = entry
	}
	for _, m := range entry.members {
		if m.AssetID == assetID && m.Value == value {
			return
		}
	}
	entry.ports[port] = struct{}{}
	entry.members = append(entry.members, Member{
		Value:     value,
		AssetID:   assetID,
		BucketKey: bucketKey,
	})
}

// Len returns the number of IPs waiting to flush.
func (a *IPPortAggregator) Len() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.byIP)
}

// FlushAll emits one flush event per accumulated IP and clears state.
func (a *IPPortAggregator) FlushAll() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for key, entry := range a.byIP {
		a.flushEntryLocked(key, entry)
	}
	a.byIP = make(map[string]*ipPortEntry)
}

func (a *IPPortAggregator) flushEntryLocked(_ string, entry *ipPortEntry) {
	if len(entry.ports) == 0 || a.onFlush == nil {
		return
	}
	a.gen++
	gen := a.gen
	path := filepath.Join(a.dataDir, fmt.Sprintf("nmap_ip_%d.txt", gen))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	if err := os.WriteFile(path, []byte(entry.ip+"\n"), 0o644); err != nil {
		return
	}
	a.onFlush(entry.ip, FlushEvent{
		Members:    append([]Member(nil), entry.members...),
		FilePath:   path,
		Generation: gen,
	})
}

// SortedPortsFromMembers extracts unique sorted ports from batch member values (ip:port).
func SortedPortsFromMembers(members []Member) []int {
	seen := make(map[int]struct{})
	for _, m := range members {
		_, port := ParseHostPort(m.Value)
		if port > 0 {
			seen[port] = struct{}{}
		}
	}
	ports := make([]int, 0, len(seen))
	for p := range seen {
		ports = append(ports, p)
	}
	sort.Ints(ports)
	return ports
}
