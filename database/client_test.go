package database_test

import (
	"testing"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/database"
)

func TestCreate(t *testing.T) {
	client := database.CreateClient()
	if client == nil {
		t.Fail()
	}
}

func TestRefCounting(t *testing.T) {
	client := database.CreateClient()
	client.Open()
	client.Open()
	client.Open()
	client.Close()
	client.Close()
	client.Close()
	client.Close()
}

type TestStruct struct {
	time     time.Time
	didOpen  bool
	refCount int
}

func TestInit(t *testing.T) {
	// _, filename, _, _ := runtime.Caller(0)
	// fmt.Println(filename)
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/mama.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()
	if !client.IsOpen() {
		t.Fail()
	}
	client.Open()
	client.Close()
	if !client.IsOpen() {
		t.Fail()
	}
	client.Close()
	if client.IsOpen() {
		t.Fail()
	}
	client.Close()
}
