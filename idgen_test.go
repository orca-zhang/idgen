package idgen

import (
	"log"
	"testing"
	"time"

	"github.com/go-redis/redis/v7"
)

var redisCli *redis.Client

func TestParse(t *testing.T) {
	redisCli = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	idg := NewIDGen(redisCli, 10)

	var ids [10000]int64
	for i := 0; i < 10000; i++ {
		ids[i], _ = idg.New()
	}

	for i := 0; i < 10000; i++ {
		ts, i, s := Parse(ids[i])
		log.Println(ids[i], time.Unix(ts, 0), i, s)
	}
}
