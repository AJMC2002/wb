package cache

import "sync/atomic"

type Stats struct {
	hits   atomic.Uint64
	misses atomic.Uint64
}

func NewStats() *Stats { return &Stats{} }

func (s *Stats) IncHit()  { s.hits.Add(1) }
func (s *Stats) IncMiss() { s.misses.Add(1) }

func (s *Stats) Snapshot() (hits uint64, misses uint64) {
	return s.hits.Load(), s.misses.Load()
}
