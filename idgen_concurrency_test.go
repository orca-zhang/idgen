package idgen

import (
	"fmt"
	"sync"
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
				cache.Del(fmt.Sprintf("idgen:3:%d", now()))
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
