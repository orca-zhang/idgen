package idgen

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestLocalConcurrentUniqueIDs(t *testing.T) {
	for _, shared := range []bool{true, false} {
		name := "shared_generator"
		if !shared {
			name = "shared_instance_across_generators"
		}
		t.Run(name, func(t *testing.T) {
			const workers, perWorker = 32, 128
			for round := 0; round < 8; round++ {
				generator := NewIDGen(nil, 3)
				start := make(chan struct{})
				ids := make(chan int64, workers*perWorker)
				var wg sync.WaitGroup
				for i := 0; i < workers; i++ {
					ig := generator
					if !shared {
						// Instance IDs are masked to four bits and share one counter.
						ig = NewIDGen(nil, int64(3+16*(i%2)))
					}
					wg.Add(1)
					go func(ig *IDGen) {
						defer wg.Done()
						<-start
						for j := 0; j < perWorker; j++ {
							id, err := ig.New()
							if err != nil {
								t.Error(err)
								return
							}
							ids <- id
						}
					}(ig)
				}
				close(start)
				wg.Wait()
				close(ids)
				seen := make(map[int64]bool, workers*perWorker)
				for id := range ids {
					if seen[id] {
						t.Fatalf("round %d: duplicate ID %d", round, id)
					}
					seen[id] = true
					_, instance, _ := Parse(id)
					if instance != 3 {
						t.Fatalf("ID %d has instance %d, want 3", id, instance)
					}
				}
				if len(seen) != workers*perWorker {
					t.Fatalf("generated %d IDs, want %d", len(seen), workers*perWorker)
				}
			}
		})
	}
}

func TestPreparedClockRollback(t *testing.T) {
	const ts = timeOff + 86400
	var first [16]*[2]int64
	for inst := range first {
		cache.Del([2]int64{ts, int64(inst)})
	}
	prepareClock(ts)
	for inst := range first {
		v, ok := cache.Get([2]int64{ts, int64(inst)})
		if !ok {
			t.Fatalf("instance %d was not prepared", inst)
		}
		first[inst] = v.(*[2]int64)
		if first[inst][0] != ts || atomic.AddInt64(&first[inst][1], 1) != 0 {
			t.Fatalf("instance %d did not start at zero", inst)
		}
	}
	for offset := int64(1); offset <= 60; offset++ {
		prepareClock(ts + offset)
	}
	prepareClock(ts)
	for inst, counter := range first {
		v, ok := cache.Get([2]int64{ts, int64(inst)})
		if !ok || v.(*[2]int64) != counter || atomic.AddInt64(&counter[1], 1) != 1 {
			t.Fatalf("instance %d: rollback reset the counter", inst)
		}
	}
}

func TestLocalRefreshCounter(t *testing.T) {
	// A stale per-generator pointer must yield to the prepared current second.
	stale := [2]int64{timeOff, 42}
	ig := NewIDGen(nil, 19)
	ig.counter.Store(&stale)
	id, err := ig.New()
	if err != nil {
		t.Fatal(err)
	}
	ts, inst, sn := Parse(id)
	counter := ig.counter.Load()
	if ts == stale[0] || inst != 3 || counter[0] != ts || stale[1] != 42 {
		t.Fatalf("stale counter was used: ID=%d, counter=%v, stale=%v", id, counter, stale)
	}
	v, ok := cache.Get([2]int64{ts, inst})
	if !ok || v.(*[2]int64) != counter {
		t.Fatal("generator did not reuse the shared cached counter")
	}
	// Repeated calls use the same counter while the published second is unchanged.
	for i := 0; i < 128; i++ {
		id, err = ig.New()
		if err != nil {
			t.Fatal(err)
		}
		nextTS, nextInst, nextSN := Parse(id)
		if nextInst != inst || (nextTS == ts && nextSN != (sn+1)&0x3FFFF) {
			t.Fatalf("sequence did not continue: previous=(%d,%d), next=(%d,%d)", ts, sn, nextTS, nextSN)
		}
		ts, sn = nextTS, nextSN
	}
}
