package test

import (
	"os"
)

const confPath = "../cmd/config"
const reportDir = "../../report-templates/templates"
const alias = "pgdemo"
const username = "admin"

var password = os.Getenv("GO_TEST_USER_PASSWORD")
