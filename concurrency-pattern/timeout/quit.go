package main

import (
	"fmt"
	"math/rand"
	"time"
)

func boring(msg string, quit <-chan bool) <-chan string { // Returns receive-only channel of strings.
	c := make(chan string)
	go func() { // We launch the goroutine from inside the function.
		for i := 0; ; i++ {
			select {
			case c <- fmt.Sprintf("%s %d", msg, i):
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
			case <-quit:
				return
			}
		}
	}()
	return c // Return the channel to the caller.
}

func main() {
	quit := make(chan bool)
	c := boring("Joe", quit)
	d := boring("Ann", quit)
	//Create the timer once, outside the loop, to time out the entire conversation.
	timeout := time.After(3 * time.Second)
	for {
		select {
		case s := <-c:
			fmt.Println(s)
		case s := <-d:
			fmt.Println(s)
		case <-timeout:
			quit <- true
			quit <- true
			fmt.Println("You talk too much.")
			return
		}
	}
}
