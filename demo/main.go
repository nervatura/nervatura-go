/*
Nervatura Demo Application
*/

package main

import (
	"github.com/nervatura/nervatura-go/demo/app"

	_ "github.com/nervatura/nervatura-client"
	_ "github.com/nervatura/nervatura-demo"
	_ "github.com/nervatura/nervatura-docs"
	_ "github.com/nervatura/report-templates"
)

func main() {

	err := app.New()
	if err != nil {
		panic(err)
	}

}
