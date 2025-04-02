package main

import (
	"log"
	"pocketbase_extend/src/hooks"

	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading environment variables:", err)
	}

	app := pocketbase.New()

	hooks.RegisterAllHooks(app)

	if err := app.Start(); err != nil {
		log.Panicln(err)
	}
}
