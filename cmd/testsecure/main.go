package main

import (
	"fmt"

	"github.com/PayRpc/Bitcoin-Sprint/pkg/secure"
)

func main() {
	ok := secure.SelfCheck()
	fmt.Printf("Secure SelfCheck: %v\n", ok)
}
