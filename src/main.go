package main

import (
	"log"
	"os"
	"path/filepath"
	"pocketbase_extend/src/hooks"
	"pocketbase_extend/src/hooks/paddle_webhooks"

	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
)

func main() {
	// Load environment variables from .env file
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)

	if err := godotenv.Load(filepath.Join(dir, ".env")); err != nil {
		log.Println("Error loading environment variables:", err)
	}

	// Load Paddle product IDs from environment variables
	paddle_webhooks.LoadProductIDs()

	// Initialize the PocketBase app
	app := pocketbase.New()
	hooks.RegisterAllHooks(app)

	if err := app.Start(); err != nil {
		log.Panicln(err)
	}
}
