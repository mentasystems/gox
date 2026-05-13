package main

import "fmt"

func transfer(userID, orderID string) {
	fmt.Printf("transfer userID=%s orderID=%s\n", userID, orderID)
}

func main() {
	// Both args are strings. The wrong order compiles fine and ships.
	transfer("o-42", "u-7")
}
