package main

import (
	"fmt"
	"math/rand"
	"time"
)

var (
	Web   = fakeSearch("web")
	Image = fakeSearch("image")
	Video = fakeSearch("video")
)

type Result string

type Search func(query string) Result

func fakeSearch(kind string) Search {
	return func(query string) Result {
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		return Result(fmt.Sprintf("%s result for %q\n", kind, query))
	}
}

//Avoid timeout
//Q: How do we avoid discarding results from slow servers?
//A: Replicate the servers. Send requests to multiple replicas, and use the first response.
func First(query string, replicas ...Search) Result {
	c := make(chan Result)
	searchReplica := func(i int) { c <- replicas[i](query) }
	for i := range replicas {
		go searchReplica(i)
	}
	//只从chanel中取了一次值，就直接返回
	return <-c
}

//Reduce tail latency using replicated search servers.
func Google(query string) (results []Result) {
	c := make(chan Result)
	go func() { c <- First(query, fakeSearch("Web1"), fakeSearch("Web2")) }()
	go func() { c <- First(query, fakeSearch("Image1"), fakeSearch("Image2")) }()
	go func() { c <- First(query, fakeSearch("Video1"), fakeSearch("Video2")) }()
	timeout := time.After(80 * time.Millisecond)
	for i := 0; i < 3; i++ {
		select {
		case result := <-c:
			results = append(results, result)
		case <-timeout:
			fmt.Println("timed out")
			return
		}
	}
	return
}

func main() {
	rand.Seed(time.Now().UnixNano())
	start := time.Now()
	result := Google("golang")
	elapsed := time.Since(start)
	fmt.Println(result)
	fmt.Println(elapsed)
}
