package main

import (
	"fmt"
	"os"

	servicecmd "kg-service/cmd"
)

func main() {
	if err := servicecmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
