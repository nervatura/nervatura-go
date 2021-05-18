package main

import (
	"fmt"

	"github.com/nervatura/nervatura-go/app"
)

var Version string

func main() {
	fmt.Printf("Version: %s\n", Version)
	app.New(Version)
}
