## 框架
Fetcher
- fetcher获取item条目以及下一次尝试获取的时间点。
- 如果获取uri条目失败返回相应error

Subscription
- 展示订阅的item条目
- 取消item的订阅

Merge
- Subscription的聚合

## 主要代码
```golang
// converts Fetches to a stream
func Subscribe(fetcher Fetcher) Subscription {
	s := &sub{
		fetcher: fetcher,
		updates: make(chan Item), // for Updates
	}
	go s.loop()
	return s
}

// sub implements the Subscription interface.
type sub struct {
	fetcher Fetcher   // fetches items
	updates chan Item // delivers items to the user
	closed  bool
	err     error
}

func (s *sub) loop() {
	for {
		if s.closed {       //①
			close(s.updates)
			return
		}
		items, next, err := s.fetcher.Fetch()
		if err != nil {
			s.err = err    //②
			time.Sleep(10 * time.Second)  //③
			continue
		}
		for _, item := range items {
			s.updates <- item             //④
		}
		if now := time.Now(); next.After(now) {
			time.Sleep(next.Sub(now))     //⑤
		}
	}
}

func (s *sub) Updates() <-chan Item {
	return s.updates
}

func (s *sub) Close() error {
	s.closed = true //⑥
	return s.err    //⑦
}
```
### loop方法

#### 1.实现的功能
1. 定期的使用s.fetcher循环获取items；
2. 将获取的items通过s.updates展示出来；
3. 当调用s.Close时退出循环。

#### 2.本实现的缺问题：
- Bug 1: unsynchronized access to s.closed/s.err
- Bug 2: time.Sleep may keep loop running
- Bug 3: loop may block forever on s.updates

//Solution

//Change the body of loop to a select with three cases:

// 1.Close was called
// 2.it's time to call Fetch
// 3.send an item on s.updates