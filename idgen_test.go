package idgen

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v7"
)

var redisCli *redis.Client

func TestNewIDGenRedis(t *testing.T) {
	redisCli = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	idgen := NewIDGen(redisCli, 0)

	for i := 0; i < 1000; i++ {
		log.Println(idgen.New())
	}
}

func TestNewIDGenLocal(t *testing.T) {
	idgen := NewIDGen(nil, 0)

	for i := 0; i < 1000; i++ {
		log.Println(idgen.New())
	}
}

func TestNewIDGenRedisConcurrent(t *testing.T) {
	redisCli = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	wg := sync.WaitGroup{}
	wg.Add(16)
	for i := 0; i < 16; i++ {
		inst := i
		go func(inst int64) {
			idgen := NewIDGen(redisCli, inst)
			for j := 0; j < 200; j++ {
				id, _ := idgen.New()

				ts, inst, sn := Parse(id)
				log.Println(id, time.Unix(ts, sn), inst, sn)
			}
			wg.Done()
		}(int64(inst))
	}
	wg.Wait()
}

func TestNewIDGenLocalConcurrent(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(16)
	for i := 0; i < 16; i++ {
		inst := i
		go func(inst int64) {
			idgen := NewIDGen(nil, inst)
			for j := 0; j < 200; j++ {
				id, _ := idgen.New()

				ts, inst, sn := Parse(id)
				log.Println(id, time.Unix(ts, sn), inst, sn)
			}
			wg.Done()
		}(int64(inst))
	}
	wg.Wait()
}

func TestParse(t *testing.T) {
	redisCli = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	idgen := NewIDGen(redisCli, 0)

	var ids [1000]int64
	for i := 0; i < 1000; i++ {
		ids[i], _ = idgen.New()
	}

	for i := 0; i < 1000; i++ {
		ts, inst, sn := Parse(ids[i])
		log.Println(ids[i], time.Unix(ts, sn), inst, sn)
	}
}
