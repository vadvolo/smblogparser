package main

import (
	"fmt"
	"os"

	"github.com/vadvolo/smblogparser/pkg/types"
)

func main(){
	fmt.Println("Hello, world")
	devArg := os.Args[1]
	logger := types.NewLogger(devArg)

	logger.ReadBytes()
	logger.ExportCVS()
}
