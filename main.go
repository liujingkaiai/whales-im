package main

import "fmt"

func main() {
	var c interface{}
	c = "sds"

	switch v := c.(type) {
	case int:
		fmt.Println(v)
	default:
		fmt.Println()
	}
}
