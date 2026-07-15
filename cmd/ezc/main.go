package main

import (
	"os"

	"github.com/mornhussakuyo-hub/ezc/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:]))
}
