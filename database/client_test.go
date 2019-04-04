package database_test

import (
	"testing"

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
	time    int64
	didOpen bool
	name    string
}

func TestInit(t *testing.T) {
	// _, filename, _, _ := runtime.Caller(0)
	// fmt.Println(filename)
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/mama.json")
	client := database.CreateClient()
	client.Init(credential)
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

func TestRegisterStruct(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	register := make([]database.DBRegisterForm, 1)
	keyCols := make([]string, 1)
	keyCols[0] = "name"
	register[0] = database.DBRegisterForm{
		BaseStruct: TestStruct{},
		Name:       "",
		SetKeys:    true,
		KeyColumns: keyCols,
	}
	client.RegisterStruct(register)
}

func TestDropTable(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	register := make([]database.DBRegisterForm, 1)
	register[0] = database.DBRegisterForm{
		BaseStruct: TestStruct{},
		Name:       "",
		SetKeys:    false,
		KeyColumns: nil,
	}
	client.RegisterStruct(register)
	client.DropTable(register)
}
