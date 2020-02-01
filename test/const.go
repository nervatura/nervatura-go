package test

import (
	"os"
)

const confPath = "../demo/config"
const reportDir = "../../report-templates/templates"
const alias = "test"
const username = "admin"

var password = os.Getenv("GO_TEST_USER_PASSWORD")
