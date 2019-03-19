package main

import (
	"fmt"
	"sync"
)

func gen(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

func sq(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			out <- n * n
		}
	}()
	return out
}

func merge(cs ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	out := make(chan int)

	// 为cs中每个输入channel启动输出Goroutine。
	// output从c中复制数值，直到c被关闭之后调用wg.Done
	output := func(c <-chan int) {
		defer wg.Done()
		for n := range c {
			out <- n
		}
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// 启动一个Goroutine，当所有output Goroutine都工作完后（wg.Done），
	// 关闭out， 且保证只关闭一次。这个Goroutine必须在wg.Add之后启动
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func main() {
	in := gen(2, 3, 4, 5, 6)

	// 在两个从in里读取数据的Goroutine间分配sq的工作
	c1 := sq(in)
	c2 := sq(in)

	// 输出从c1和c2合并的数据
	for n := range merge(c1, c2) {
		fmt.Println(n) // 4 和 9, 或者 9 和 4
	}
}
