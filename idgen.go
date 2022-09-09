package idgen

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/orca-zhang/ecache"
)

const timeOff = 1662688799 // 2022-09-09 09:59:59, js integer won't overflow(53bit) before 2090-09-27 13:14:06(3810172446)
const expiration = time.Minute

var cache = ecache.NewLRUCache(16, 32, expiration)
var clock = time.Now().Unix()

func now() int64 { return atomic.LoadInt64(&clock) }
func init() {
	rand.Seed(now())
	go func() { // internal counter that reduce GC caused by `time.Now()`
		for {
			atomic.StoreInt64(&clock, time.Now().Unix()) // calibration every second
			time.Sleep(300 * time.Millisecond)
		}
	}()
}

type IDGen struct {
	redisCli redis.Cmdable
	instID   int64
}

func NewIDGen(redis redis.Cmdable, inst int64) *IDGen {
	return &IDGen{redisCli: redis, instID: inst}
}

func (ig *IDGen) New() (id int64, err error) {
	ts, sn := now(), int64(0)
	key := fmt.Sprintf("idgen:%d:%d", (ig.instID & 0xF), ts)
	if ig.redisCli != nil {
		if sn, err = ig.redisCli.Incr(key).Result(); err != nil {
			sn = rand.Int63n(131072) | 0x200000 // downgrade to use random num
		} else if sn == 1 {
			ig.redisCli.Expire(key, expiration) // new item, set expiration
		}
	} else {
		if v, ok := cache.Get(key); ok {
			sn = atomic.AddInt64(v.(*int64), 1)
		} else {
			cache.Put(key, &sn)
		}
	}
	return ((ts - timeOff) << 22) + ((ig.instID & 0xF) << 18) + (sn & 0x3FFFF), err
}

func Parse(id int64) (ts int64, instID int64, sn int64) {
	return (id >> 22) + timeOff, (id >> 18) & 0xF, id & 0x3FFFF
}
