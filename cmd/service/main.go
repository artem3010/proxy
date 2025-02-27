package main

import (
	"os"
	"proxy/internal/app"
)

func main() {
	application := app.New()
	os.Exit(application.Run())
}
