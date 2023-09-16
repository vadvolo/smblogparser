package main

import (
	"fmt"

	"github.com/vadvolo/smblogparser/pkg/types"
)

func main(){
	fmt.Println("Hello, world")
	logger := types.NewLogger

	logger().ReadBytes()
}
