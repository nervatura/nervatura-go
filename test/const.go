package test

import (
	"os"
)

const confPath = "../demo/config"
const alias = "test"
const username = "admin"
const testToken = "eyJhbGciOiJIUzI1NiIsImtpZCI6IjhCRDBFMDI1MDk0ODJGNThDRUZEM0MwRUNENDFGRjBFIiwidHlwIjoiSldUIn0.eyJ1c2VybmFtZSI6ImFkbWluIiwiZGF0YWJhc2UiOiJ0ZXN0IiwiZXhwIjoxNTgwNTI0NjQyLCJpc3MiOiJuZXJ2YXR1cmEifQ.tzmV1osKO702VSFKFsGLekpAHL_EpafH1G7e9StKs0E"

var password = os.Getenv("GO_TEST_USER_PASSWORD")
