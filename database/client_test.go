package database_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
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
	Time    int64
	DidOpen bool
	Name    string
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
	keyCols[0] = "Time"
	register[0] = database.DBRegisterForm{
		BaseStruct: TestStruct{},
		Name:       "",
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
		KeyColumns: nil,
	}
	client.RegisterStruct(register)
	client.DropTable(register)
}

func TestInsert(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	register := make([]database.DBRegisterForm, 1)
	keyCols := make([]string, 1)
	keyCols[0] = "Time"
	register[0] = database.DBRegisterForm{
		BaseStruct: TestStruct{},
		Name:       "",
		KeyColumns: keyCols,
	}
	client.RegisterStruct(register)

	t1 := TestStruct{Time: time.Now().UnixNano()}
	t2 := TestStruct{Time: time.Now().UnixNano() + 1}
	ok, err := client.Insert(&t1, &t2)
	if !ok {
		logger.Error("Insert failed: %s", err.Error())
		t.Fail()
	}
}

func TestSelect(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	register := make([]database.DBRegisterForm, 1)
	keyCols := make([]string, 1)
	keyCols[0] = "Time"
	register[0] = database.DBRegisterForm{
		BaseStruct: TestStruct{},
		Name:       "",
		KeyColumns: keyCols,
	}
	client.RegisterStruct(register)

	t1 := TestStruct{Time: time.Now().UnixNano(), Name: "Meme", DidOpen: false}
	t2 := TestStruct{Time: time.Now().UnixNano() + 10, Name: "Neme", DidOpen: true}
	t3 := TestStruct{Time: time.Now().UnixNano() + 20, Name: "Meme", DidOpen: true}
	t4 := TestStruct{Time: time.Now().UnixNano() + 30, Name: "Neme", DidOpen: false}

	ok, err := client.Insert(&t1, &t2, &t3, &t4)
	if !ok {
		logger.Error("Insert failed: %s", err.Error())
		t.Fail()
	}

	var bucket []TestStruct
	ok, err = client.Select(&bucket, "select * from TestStruct where Name=? and DidOpen=? order by Time", "Meme", true)
	if !ok {
		logger.Error("Select failed: %s", err.Error())
		t.Fail()
	}
	if len(bucket) != 1 {
		logger.Error("Select failed: should have selected 1, got %d", len(bucket))
		t.Fail()
	}
	for _, v := range bucket {
		logger.Info("%v", v)
	}
}

func TestDelete(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	register := make([]database.DBRegisterForm, 1)
	keyCols := make([]string, 1)
	keyCols[0] = "Time"
	register[0] = database.DBRegisterForm{
		BaseStruct: TestStruct{},
		Name:       "",
		KeyColumns: keyCols,
	}
	client.RegisterStruct(register)

	ok, err := client.Delete(TestStruct{}, "where DidOpen=?", true)
	if !ok {
		logger.Error("Delete failed: %s", err.Error())
		t.Fail()
	}
}

func TestDeleteWrongQuery(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	register := make([]database.DBRegisterForm, 1)
	keyCols := make([]string, 1)
	keyCols[0] = "Time"
	register[0] = database.DBRegisterForm{
		BaseStruct: TestStruct{},
		Name:       "",
		KeyColumns: keyCols,
	}
	client.RegisterStruct(register)

	ok, err := client.Delete(TestStruct{}, "    somethingweirdo where DidOpen=?", true)
	if ok {
		logger.Error("Test Failed")
	}
	logger.Error("Test Success: %s", err.Error())
}

func TestSelectNull(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	register := make([]database.DBRegisterForm, 1)
	keyCols := make([]string, 1)
	keyCols[0] = "Time"
	register[0] = database.DBRegisterForm{
		BaseStruct: TestStruct{},
		Name:       "",
		KeyColumns: keyCols,
	}
	client.RegisterStruct(register)

	var bucket []TestStruct
	fmt.Println(bucket)
	fmt.Println(len(bucket))
	fmt.Println(bucket == nil)
	ok, err := client.Select(&bucket, "select * from TestStruct where Time=(select max(Time) from TestStruct where Name=?)", "Pepe")
	if !ok {
		logger.Error("Select failed: %s", err.Error())
		t.Fail()
	}
	for _, v := range bucket {
		logger.Info("%v", v)
	}
}

func TestUpsert(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	register := make([]database.DBRegisterForm, 1)
	keyCols := make([]string, 1)
	keyCols[0] = "Time"
	register[0] = database.DBRegisterForm{
		BaseStruct: TestStruct{},
		Name:       "",
		KeyColumns: keyCols,
	}
	client.RegisterStruct(register)

	t1 := TestStruct{Time: time.Now().UnixNano(), Name: "Meme", DidOpen: false}
	t2 := TestStruct{Time: time.Now().UnixNano(), Name: "Nemp", DidOpen: true}

	_, err := client.Upsert(&t1)
	_, err = client.Upsert(&t2)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Test Success")
	}
}

func TestBulkInsert(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	register := make([]database.DBRegisterForm, 1)
	keyCols := make([]string, 1)
	keyCols[0] = "Time"
	register[0] = database.DBRegisterForm{
		BaseStruct: TestStruct{},
		Name:       "",
		KeyColumns: keyCols,
	}
	client.RegisterStruct(register)

	stress := 5000

	testdata := make([]interface{}, stress)
	for i := 0; i < stress; i++ {
		testdata[i] = &TestStruct{Time: int64(i), Name: ("Meme" + strconv.Itoa(i)), DidOpen: i%2 == 0}
	}

	client.Delete(TestStruct{}, "where Time < ?", stress)
	countTime := func(f func()) int64 {
		start := time.Now()
		f()
		return time.Now().UnixNano() - start.UnixNano()
	}

	timeBulk := countTime(func() {
		_, err := client.BulkInsert(testdata...)
		if err != nil {
			fmt.Println(err.Error())
		}
	})
	fmt.Printf("Time Bulk: %f\n", float64(timeBulk)/float64(time.Second))

	// client.Delete(TestStruct{}, "where Time < ?", stress)

	// timeInsert := countTime(func() {
	// 	_, err := client.Insert(testdata...)
	// 	if err != nil {
	// 		fmt.Println(err.Error())
	// 	}
	// })
	// fmt.Printf("Time Insert: %f\n", float64(timeInsert)/float64(time.Second))

}
