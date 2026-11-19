package idgen

import (
	"sync/atomic"
	"testing"
)

var benchmarkIDs atomic.Int64

// These measure call cost; the 18-bit sequence limit still applies.
func BenchmarkNewLocalSerial(b *testing.B) {
	ig := NewIDGen(nil, 0)
	var ids int64
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := ig.New()
		if err != nil {
			b.Fatal(err)
		}
		ids ^= id
	}
	benchmarkIDs.Store(ids)
}

func BenchmarkNewLocalShared(b *testing.B) {
	ig := NewIDGen(nil, 0)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var ids int64
		for pb.Next() {
			id, err := ig.New()
			if err != nil {
				b.Fatal(err)
			}
			ids ^= id
		}
		benchmarkIDs.Store(ids)
	})
}

func BenchmarkNewLocalInstances(b *testing.B) {
	var instance atomic.Int64
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		ig := NewIDGen(nil, instance.Add(1)-1)
		var ids int64
		for pb.Next() {
			id, err := ig.New()
			if err != nil {
				b.Fatal(err)
			}
			ids ^= id
		}
		benchmarkIDs.Store(ids)
	})
}

func BenchmarkNewLocalFresh(b *testing.B) {
	var ids int64
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := NewIDGen(nil, 0).New()
		if err != nil {
			b.Fatal(err)
		}
		ids ^= id
	}
	benchmarkIDs.Store(ids)
}
