package database_test

import (
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/database"
)

func TestCreate(t *testing.T) {
	client := database.Create()
	if client == nil {
		t.Fail()
	}
}

func TestRefCounting(t *testing.T) {
	client := database.Create()
	client.Open()
	client.Open()
	client.Open()
	client.Close()
	client.Close()
	client.Close()
	client.Close()
}
