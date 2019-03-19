package main

/*
 * 1. Channel both communicate and synchronize
 * 2. Go channels can also be created with a buffer.
 *    Buffered channel removes synchronization.
 */

import (
	"fmt"
	"math/rand"
	"time"
)

// Returns receive-only channel of strings.
// Channels are first-class values, just like strings or integers.
func boring(msg string) <-chan string {
	c := make(chan string)
	go func() { // We launch the goroutine from inside the function.
		for i := 0; ; i++ {
			c <- fmt.Sprintf("%s %d", msg, i)
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		}
	}()

	return c // Return the channel to the caller.
}

// fanIn-Pattern using select
func fanIn(input1, input2 <-chan string) <-chan string {
	c := make(chan string)
	go func() {
		for {
			select {
			case s := <-input1:
				c <- s
			case s := <-input2:
				c <- s
			}
		}
	}()
	return c
}

func main() {
	c := fanIn(boring("Joe"), boring("Ann"))
	for i := 0; i < 10; i++ {
		fmt.Println(<-c)
	}

	fmt.Println("You're both boring; I'm leaving.")
}
