// naivemain runs the Subscribe example with the naive Subscribe
// implementation and a fake RSS fetcher.
package main

import (
	"fmt"
	"math/rand"
	"time"

	"goTools/advanced-concurrency/feedReader/naive"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	// Subscribe to some feeds, and create a merged update stream.
	merged := naive.Merge(
		naive.NaiveSubscribe(naive.Fetch("blog.golang.org")),
		naive.NaiveSubscribe(naive.Fetch("googleblog.blogspot.com")),
		naive.NaiveSubscribe(naive.Fetch("googledevelopers.blogspot.com")))

	// Close the subscriptions after some time.
	time.AfterFunc(3*time.Second, func() {
		fmt.Println("closed:", merged.Close())
	})

	// Print the stream.
	for it := range merged.Updates() {
		fmt.Println(it.Channel, it.Title)
	}

	// The loops are still running.  Let the race detector notice.
	time.Sleep(1 * time.Second)

	panic("show me the stacks")
}
