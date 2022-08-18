package idgen

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/orca-zhang/ecache"
)

const timeOff = 1660817898 // 2022-08-18 18:18:18, js integer won't overflow(53bit) before 2039-08-23 13:06:49(2197688809)
const expiration = time.Minute

var cache = ecache.NewLRUCache(16, 32, expiration)
var clock = time.Now().Unix()

func now() int64 { return atomic.LoadInt64(&clock) }
func init() {
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

func init() {
	rand.Seed(time.Now().Unix())
}

func NewIDGen(redis redis.Cmdable, i int64) *IDGen {
	return &IDGen{redisCli: redis, instID: i}
}

func (ig *IDGen) New() (id int64, local bool) {
	ts, sn := int64(now()), int64(0)
	key := fmt.Sprintf("idgen:%d", ts)
	if ig.redisCli != nil {
		var err error
		if sn, err = ig.redisCli.Incr(key).Result(); err != nil {
			sn = rand.Int63n(1048576) // upgrade to use random num
			local = true
		} else if sn == 1 {
			ig.redisCli.Expire(key, expiration) // new item, set expiration
		}
	} else {
		if v, ok := cache.Get(key); ok {
			sn = atomic.AddInt64(v.(*int64), 1)
		} else {
			cache.Put(key, &sn)
		}
		local = true
	}
	return ((ts - timeOff) << 24) + (ig.instID&0xF)<<20 + (sn & 0xFFFFF), local
}

func Parse(id int64) (ts int64, instID int64, sn int64) {
	return (id >> 24) + timeOff, (id >> 20) & 0xF, id & 0xFFFFF
}