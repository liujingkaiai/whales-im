package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	for i := 1; i <= 10000; i++ {
		go func(i int) {
			fmt.Println(i)
		}(i)
	}
	time.Sleep(10 * time.Second)
	log.Println("all req done")
}
