package paddle_webhooks

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

const testDataDir = "../../../test_pb_data"

// Load environment variables from .env file for all tests in this package
func TestMain(m *testing.M) {
	if err := godotenv.Load("../../../.env"); err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}

	LoadProductIDs()

	exitCode := m.Run()
	os.Exit(exitCode)
}
