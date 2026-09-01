package postgres_tests

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../../../../.env"); err != nil {
		log.Printf("warning: could not load .env: %v", err)
	}

	os.Exit(m.Run())
}