# idgen

<p align="center">一款基于❄️算法实现的🆔生成器</p>
<p align="center">
  <a href="/go.mod#L3" alt="go version">
    <img src="https://img.shields.io/badge/go%20version-%3E=1.22-brightgreen?style=flat"/>
  </a>
  <a href="https://github.com/orca-zhang/ecache/blob/master/LICENSE" alt="license MIT">
    <img src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat">
  </a>
</p>

## 特性

- 🚀 支持基于redis或者本地生成（redis操作失败降级为随机数）
- 🔒 本地模式支持并发调用；同一进程内相同实例号的生成器共享序号
- ⌚ ntp同步和时钟回跳安全（默认最多`expiration`时间内，目前为一分钟）
- 🦖 js对接时整型精度不丢失，`2090-09-27 13:14:06(3810172446)`前不会超过53位
- 🏳️‍🌈 42位时间戳+4位实例号（可多个节点复用，最多可独立部署16个发号器）+18位序号（一秒内单实例分配不超过13万个）

本地发号的实现与性能数据见 [计数器缓存行优化](benchmarks/2026-11-19/counter-padding/README.md)。

## 如何使用

#### 引入包（预计5秒）
``` go
import (
    "github.com/orca-zhang/idgen"
)
```

#### 定义实例（预计5秒）
> 可以放置在任意位置（全局也可以），建议就近定义
``` go
var ig = idgen.NewIDGen(redisCli, 0) // 参数1是redis连接，传nil说明是本地生成，参数2是实例号(会取模16)
```

#### 获取🆔（预计5秒）
``` go
id, err := ig.New() // 返回生成的id，以及是否出错
                    //（只有在redis出错的情况下才会返回err，sn部分用的是随机数，最高位是1）
                    // 有错误时可忽略，id依然可用，此时是降级
```

#### 解析🆔（预计5秒）
``` go
ts, inst, sn := idgen.Parse(id) // 返回`秒级时间戳`，`实例号`，`序列号`
```

#### 下载包（预计5秒）

> 非go modules模式：\
> sh>  ```go get -u github.com/orca-zhang/idgen```

> go modules模式：\
> sh>  ```go mod tidy && go mod download```

#### 运行吧
> 🎉 完美搞定
> sh>  ```go run <你的main.go文件>```
