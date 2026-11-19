package idgen

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/orca-zhang/ecache2"
	"golang.org/x/sys/cpu"
)

const timeOff = 1662688799 // 2022-09-09 09:59:59, js integer won't overflow(53bit) before 2090-09-27 13:14:06(3810172446)
const expiration = time.Minute

// One entry per shard per update; 256 retains a minute of 300 ms updates.
var cache = ecache2.NewLRUCache[[2]int64](16, 256, expiration)
var clock = time.Now().Unix()

func now() int64 { return atomic.LoadInt64(&clock) }

// Only the clock updater creates counters, before publishing the timestamp.
func prepareClock(ts int64) {
	for inst := int64(0); inst < 16; inst++ {
		key := [2]int64{ts, inst}
		if _, ok := cache.Get(key); !ok {
			// Different instances must not share the sequence's cache line.
			counter := struct {
				value [2]int64
				_     cpu.CacheLinePad
			}{value: [2]int64{ts, -1}}
			cache.Put(key, &counter.value)
		}
	}
}

func init() {
	prepareClock(now())
	rand.Seed(now())
	go func() { // internal counter that reduce GC caused by `time.Now()`
		for {
			ts := time.Now().Unix()
			prepareClock(ts)
			atomic.StoreInt64(&clock, ts)
			time.Sleep(300 * time.Millisecond)
		}
	}()
}

// IDGen must not be copied after first use.
type IDGen struct {
	redisCli redis.Cmdable
	instID   int64
	counter  atomic.Pointer[[2]int64] // current [timestamp, sequence], shared through cache
}

func NewIDGen(redis redis.Cmdable, inst int64) *IDGen {
	return &IDGen{redisCli: redis, instID: inst}
}

func (ig *IDGen) New() (id int64, err error) {
	ts, sn := now(), int64(0)
	if ig.redisCli != nil {
		key := fmt.Sprintf("idgen:%d:%d", ig.instID&0xF, ts)
		if sn, err = ig.redisCli.Incr(key).Result(); err != nil {
			sn = rand.Int63n(131072) | 0x200000 // downgrade to use random num
		} else if sn == 1 {
			ig.redisCli.Expire(key, expiration) // new item, set expiration
		}
	} else {
		counter := ig.counter.Load()
		for counter == nil || counter[0] != ts {
			if v, ok := cache.Get([2]int64{ts, ig.instID & 0xF}); ok {
				counter = v.(*[2]int64)
				ig.counter.Store(counter)
				break
			}
			runtime.Gosched() // allow clock preparation after a pause longer than TTL
			ts = now()
		}
		sn = atomic.AddInt64(&counter[1], 1)
	}
	return ((ts - timeOff) << 22) + ((ig.instID & 0xF) << 18) + (sn & 0x3FFFF), err
}

func Parse(id int64) (ts int64, instID int64, sn int64) {
	return (id >> 22) + timeOff, (id >> 18) & 0xF, id & 0x3FFFF
}
