package presentation_test

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

// TestMain prepares the environment required by the presentation
// integration tests.
//
// The integration tests use the same local PostgreSQL configuration as the
// rest of the Identity integration-test suite.
func TestMain(m *testing.M) {
	if err := godotenv.Load("../../../.env"); err != nil {
		log.Printf("warning: could not load .env: %v", err)
	}

	os.Exit(m.Run())
}
