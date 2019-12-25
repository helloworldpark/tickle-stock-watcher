package database_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
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

	// t1 := TestStruct{Time: time.Now().UnixNano(), Name: "Meme", DidOpen: false}
	// t2 := TestStruct{Time: time.Now().UnixNano() + 10, Name: "Neme", DidOpen: true}
	// t3 := TestStruct{Time: time.Now().UnixNano() + 20, Name: "Meme", DidOpen: true}
	// t4 := TestStruct{Time: time.Now().UnixNano() + 30, Name: "Neme", DidOpen: false}

	// ok, err := client.Insert(&t1, &t2, &t3, &t4)
	// if !ok {
	// 	logger.Error("Insert failed: %s", err.Error())
	// 	t.Fail()
	// }

	var bucket []TestStruct
	ok, err := client.Select(&bucket, "where Name=? and DidOpen=? order by Time", "Meme", true)
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
	ok, err := client.Select(&bucket, "where Time=(select max(Time) from TestStruct where Name=?)", "Pepe")
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

	client.RegisterStructFromRegisterables([]database.DBRegisterable{structs.UserStock{}})

	t1 := structs.UserStock{UserID: 1111, StockID: "000000", OrderSide: 1, Repeat: false}
	t2 := structs.UserStock{UserID: 1111, StockID: "000000", OrderSide: 1, Repeat: true}

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
		_, err := client.BulkInsert(false, testdata...)
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

func TestQuery(t *testing.T) {
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
		testdata[i] = TestStruct{Time: int64(i), Name: ("Pepe" + strconv.Itoa(i)), DidOpen: i%2 == 0}
	}

	_, err := client.Upsert(testdata...)
	if err != nil {
		fmt.Println("Upsert: " + err.Error())
	}

	var bucket []TestStruct
	_, err = client.Select(&bucket, "where Time<=?", 10)
	if err != nil {
		fmt.Println("Select: " + err.Error())
	}
	for i := range bucket {
		fmt.Println(bucket[i])
	}
}

func TestExtractType(t *testing.T) {
	// var v1 []TestStruct
	// t1, err1 := database.ExtractStructType(reflect.TypeOf(v1))
	// t2, err2 := database.ExtractStructType(reflect.TypeOf(&v1))
	// var v3 []*TestStruct
	// t3, err3 := database.ExtractStructType(reflect.TypeOf(&v3))
	// var v4 int
	// t4, err4 := database.ExtractStructType(reflect.TypeOf(&v4))
	// var v5 func()
	// t5, err5 := database.ExtractStructType(reflect.TypeOf(&v5))

	// fmt.Println(t1, err1)
	// fmt.Println(t2, err2)
	// fmt.Println(t3, err3)
	// fmt.Println(t4, err4)
	// fmt.Println(t5, err5)
}

func TestAllSameType(t *testing.T) {
	v1 := make([]interface{}, 3)
	v1[0] = TestStruct{}
	v1[1] = TestStruct{}
	v1[2] = TestStruct{}
	v2 := make([]interface{}, 3)
	v2[0] = TestStruct{}
	v2[1] = struct{}{}
	v2[2] = 3

	// fmt.Println(database.IsAllSameType(v1))
	// fmt.Println(database.IsAllSameType(v2))
}

type TestKeyStruct struct {
	Time    int64
	DidOpen bool
	Name    string
}

func TestBulkUpsert(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	register := make([]database.DBRegisterForm, 1)
	register[0] = database.DBRegisterForm{
		BaseStruct: TestKeyStruct{},
		Name:       "",
		KeyColumns: []string{"Time"},
	}
	client.RegisterStruct(register)

	stress := 5000
	testdata := make([]interface{}, stress)
	for i := 0; i < stress; i++ {
		testdata[i] = TestKeyStruct{Time: int64(i) % 10, Name: ("Pepe" + strconv.Itoa(i)), DidOpen: i%2 == 0}
	}

	_, err := client.BulkUpsert(testdata...)
	if err != nil {
		fmt.Println("BulkUpsert: " + err.Error())
	}

	var bucket []TestKeyStruct
	_, err = client.Select(&bucket, "where Time<=?", 10)
	if err != nil {
		fmt.Println("Select: " + err.Error())
	}
	for i := range bucket {
		fmt.Println(bucket[i])
	}
}
