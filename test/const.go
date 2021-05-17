package test

import (
	"os"
)

const alias = "test"
const username = "admin"

var password = os.Getenv("GO_TEST_USER_PASSWORD")
var apiKey = os.Getenv("NT_API_KEY")
