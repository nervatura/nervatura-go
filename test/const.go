package test

import (
	"os"
)

const confPath = "../demo/config"
const alias = "test"
const username = "admin"
const testToken = "eyJhbGciOiJIUzI1NiIsImtpZCI6IjhCRDBFMDI1MDk0ODJGNThDRUZEM0MwRUNENDFGRjBFIiwidHlwIjoiSldUIn0.eyJ1c2VybmFtZSI6ImFkbWluIiwiZGF0YWJhc2UiOiJ0ZXN0IiwiZXhwIjoxNTgwNDQwMjYyLCJpc3MiOiJuZXJ2YXR1cmEifQ.Tbsl2VdOyCE-67dHG1z6qHbpdt5IK2boQCdkmOkY49o"

var password = os.Getenv("GO_TEST_USER_PASSWORD")
