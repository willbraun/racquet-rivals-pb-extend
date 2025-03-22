package main

import (
	"log"
	"pocketbase_extend/src/hooks"

	"github.com/pocketbase/pocketbase"
)

func main() {
	app := pocketbase.New()

	hooks.RegisterAllHooks(app)

	if err := app.Start(); err != nil {
		log.Panicln(err)
	}
}
