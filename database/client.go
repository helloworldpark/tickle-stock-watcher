package database

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"sync"

	"github.com/go-gorp/gorp"
	_ "github.com/go-sql-driver/mysql" // SQL Connection
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

// DBClient is a handle to database
type DBClient struct {
	ref        *commons.Ref
	mutex      sync.Mutex
	db         *sql.DB
	dbmap      *gorp.DbMap
	credential DBCredential
}

// DBCredential contains info needed to connect to DB
type DBCredential struct {
	InstanceConnectionName string `json:"instanceConnectionName"`
	DatabaseUser           string `json:"databaseUser"`
	Password               string `json:"password"`
	DBName                 string `json:"dbName"`
}

// DBAccess implement this to show that the struct has access to database
type DBAccess interface {
	AccessDB() *DBClient
}

// LoadCredential load DB credential from json file
func LoadCredential(filePath string) DBCredential {
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		logger.Panic("%v", err)
	}

	var cred DBCredential
	if err := json.Unmarshal(raw, &cred); err != nil {
		logger.Panic("%v", err)
	}
	return cred
}

// CreateClient creates DBClient struct
func CreateClient() *DBClient {
	ref := commons.Ref{}
	client := DBClient{ref: &ref}
	return &client
}

// Init prepares for opening database
func (client *DBClient) Init(credential DBCredential) {
	client.mutex.Lock()
	client.credential = credential
	client.mutex.Unlock()
}

// openingQuery generates SQL query for connection to database
func openingQuery(credential DBCredential) string {
	var buf bytes.Buffer
	buf.WriteString(credential.DatabaseUser)
	buf.WriteByte(':')
	buf.WriteString(credential.Password)
	buf.WriteByte('@')
	buf.WriteByte('(')
	buf.WriteString("127.0.0.1:3306")
	buf.WriteByte(')')
	buf.WriteByte('/')
	buf.WriteString(credential.DBName)
	return buf.String()
}

// Open may start a connection to database if there are no active connections.
// Calling this method many times does not have any effect to overconnection.
func (client *DBClient) Open() {
	defer client.mutex.Unlock()

	client.mutex.Lock()
	client.ref.Retain()
	if client.ref.Count() == 1 {
		openingQ := openingQuery(client.credential)
		db, err := sql.Open("mysql", openingQ)
		if err != nil {
			logger.Panic("%v", err)
			return
		}

		err = db.Ping()
		if err != nil {
			logger.Panic("%v", err)
		}

		logger.Info("[DB] Connected")
		client.db = db

		dbmap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"}}
		if dbmap == nil {
			logger.Panic("DBMap is nil")
		}
		client.dbmap = dbmap
	}
}

// Close closes connection to database only if there are no more needs for connection.
func (client *DBClient) Close() {
	defer client.mutex.Unlock()

	client.mutex.Lock()
	if client.ref.Count() <= 0 {
		return
	}
	client.ref.Release()
	if client.ref.Count() == 0 {
		client.db.Close()
		client.db = nil
		client.dbmap = nil
	}
}

// IsOpen returns true if database connection is open
func (client *DBClient) IsOpen() bool {
	defer client.mutex.Unlock()

	client.mutex.Lock()
	return client.ref.Count() > 0
}

// DBRegisterForm is a struct for registering struct type as a DB table
type DBRegisterForm struct {
	BaseStruct    interface{}
	Name          string
	AutoIncrement bool
	KeyColumns    []string
	UniqueColumns []string
}

// DBRegisterable is an interface every struct which should be recorded in database should implement
type DBRegisterable interface {
	// GetDBRegisterForm returns a DBRegisterForm struct
	GetDBRegisterForm() DBRegisterForm
}

// RegisterStruct registers struct types to gorp.DbMap
// Use this method as such:
//  	register := make([]DBRegisterForm{}, 2)
// 		register[0] = Stock{}
//      register[1] = User{}
// 		RegisterStruct(register)
func (client *DBClient) RegisterStruct(forms []DBRegisterForm) {
	if !client.IsOpen() {
		return
	}

	client.mutex.Lock()
	for _, form := range forms {
		table := client.dbmap.AddTableWithName(form.BaseStruct, form.Name)
		if len(form.KeyColumns) > 0 {
			table.SetKeys(form.AutoIncrement, form.KeyColumns...)
		}
		if len(form.UniqueColumns) > 1 {
			table.SetUniqueTogether(form.UniqueColumns...)
		}
	}
	err := client.dbmap.CreateTablesIfNotExists()
	if err != nil {
		logger.Error("Creating table failed: %s", err.Error())
	} else {
		logger.Info("Created table")
	}
	client.mutex.Unlock()
}

// RegisterStructFromRegisterables registers structs from slice of DBRegisterable
func (client *DBClient) RegisterStructFromRegisterables(registerables []DBRegisterable) {
	if !client.IsOpen() {
		return
	}

	client.mutex.Lock()
	for _, r := range registerables {
		form := r.GetDBRegisterForm()
		table := client.dbmap.AddTableWithName(form.BaseStruct, form.Name)
		if form.KeyColumns != nil && len(form.KeyColumns) > 0 {
			table.SetKeys(form.AutoIncrement, form.KeyColumns...)
		}
		if form.UniqueColumns != nil && len(form.UniqueColumns) > 1 {
			table.SetUniqueTogether(form.UniqueColumns...)
		}
	}
	err := client.dbmap.CreateTablesIfNotExists()
	if err != nil {
		logger.Error("Creating table failed: %s", err.Error())
	} else {
		logger.Info("Created table")
	}
	client.mutex.Unlock()
}

// DropTable drops table of struct if exists
func (client *DBClient) DropTable(forms []DBRegisterForm) {
	if !client.IsOpen() {
		return
	}
	var err error
	client.mutex.Lock()
	for _, form := range forms {
		err = client.dbmap.DropTableIfExists(form.BaseStruct)
		if err != nil {
			logger.Error("Dropping table failed: %s", err.Error())
		} else {
			logger.Info("Dropped table %s", reflect.TypeOf(form.BaseStruct).Name())
		}
	}
	client.mutex.Unlock()
}

type dbError struct {
	msg string
}

func (err *dbError) Error() string {
	return "[DB] " + err.msg
}

// Insert inserts struct to database
// Returns (false, nonnil error) if something wrong has happened
func (client *DBClient) Insert(o ...interface{}) (bool, error) {
	if !client.IsOpen() {
		return false, &dbError{msg: "Database is not open yet"}
	}

	var err error
	client.mutex.Lock()
	err = client.dbmap.Insert(o...)
	client.mutex.Unlock()

	return err == nil, err
}

// BulkInsert inserts data by bulk, i.e. in a one query.
// It may fail if any one of the data has a problem, i.e. all or none.
func (client *DBClient) BulkInsert(o ...interface{}) (bool, error) {
	if !client.IsOpen() {
		return false, &dbError{msg: "Database is not open yet"}
	}

	query, args, err := client.queryInsert(o, false)
	if err != nil {
		return false, err
	}

	client.mutex.Lock()
	_, err = client.dbmap.Exec(query, args...)
	client.mutex.Unlock()

	return err == nil, err
}

// Update updates value to the database
func (client *DBClient) Update(o ...interface{}) (bool, error) {
	if !client.IsOpen() {
		return false, &dbError{msg: "Database is not open yet"}
	}

	client.mutex.Lock()
	_, err := client.dbmap.Update(o...)
	client.mutex.Unlock()
	return err == nil, err
}

// Upsert performs update if the data is already inserted.
func (client *DBClient) Upsert(o ...interface{}) (bool, error) {
	if !client.IsOpen() {
		return false, &dbError{msg: "Database is not open yet"}
	}

	query, args, err := client.queryInsert(o, true)
	if err != nil {
		return false, err
	}

	client.mutex.Lock()
	_, err = client.dbmap.Exec(query, args...)
	client.mutex.Unlock()
	return err == nil, err
}

func (client *DBClient) queryInsert(o []interface{}, handleDuplicate bool) (string, []interface{}, error) {
	if len(o) == 0 {
		return "", nil, &dbError{msg: "Argument 'o' is empty"}
	}
	if !isAllSameType(o) {
		return "", nil, &dbError{msg: "Every element of slice must be of same type"}
	}
	targetType, err := extractStructType(o)
	if err != nil {
		return "", nil, err
	}

	table, err := client.dbmap.TableFor(targetType, false)
	if err != nil {
		return "", nil, err
	}

	// Generate Query String
	queryBuffer := strings.Builder{}
	queryBuffer.WriteString("insert into ")
	queryBuffer.WriteString(table.TableName)

	argsBuffer := strings.Builder{}
	valsBuffer := strings.Builder{}

	argsBuffer.WriteString(" (")
	valsBuffer.WriteString(" (")

	for i := 0; i < targetType.NumField(); i++ {
		if i > 0 {
			argsBuffer.WriteString(", ")
			valsBuffer.WriteString(", ")
		}
		fieldType := targetType.Field(i).Name
		colname := table.ColMap(fieldType).ColumnName
		argsBuffer.WriteString(colname)
		valsBuffer.WriteString("?")
	}
	argsBuffer.WriteString(")")
	valsBuffer.WriteString(")")

	queryBuffer.WriteString(argsBuffer.String())
	queryBuffer.WriteString(" values")

	// Fill in arguments, too
	args := make([]interface{}, targetType.NumField()*len(o))
	for i := range o {
		if i > 0 {
			queryBuffer.WriteString(", ")
		}
		queryBuffer.WriteString(valsBuffer.String())

		v := reflect.ValueOf(o[i])
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		for j := 0; j < targetType.NumField(); j++ {
			args[i*targetType.NumField()+j] = v.Field(j).Interface()
		}
	}

	if handleDuplicate {
		queryBuffer.WriteString(" on duplicate key update ")

		upsBuffer := strings.Builder{}
		for i := 0; i < targetType.NumField(); i++ {
			if i > 0 {
				upsBuffer.WriteString(", ")
			}
			fieldType := targetType.Field(i).Name
			colname := table.ColMap(fieldType).ColumnName
			upsBuffer.WriteString(colname)
			upsBuffer.WriteString("=values(")
			upsBuffer.WriteString(colname)
			upsBuffer.WriteString(")")
		}
		queryBuffer.WriteString(upsBuffer.String())
	}

	return queryBuffer.String(), args, nil
}

var nonStructElement = map[reflect.Kind]bool{
	reflect.Invalid:       true,
	reflect.Bool:          true,
	reflect.Int:           true,
	reflect.Int8:          true,
	reflect.Int16:         true,
	reflect.Int32:         true,
	reflect.Int64:         true,
	reflect.Uint:          true,
	reflect.Uint8:         true,
	reflect.Uint16:        true,
	reflect.Uint32:        true,
	reflect.Uint64:        true,
	reflect.Uintptr:       true,
	reflect.Float32:       true,
	reflect.Float64:       true,
	reflect.Complex64:     true,
	reflect.Complex128:    true,
	reflect.Array:         false,
	reflect.Chan:          false,
	reflect.Func:          true,
	reflect.Interface:     false,
	reflect.Map:           true,
	reflect.Ptr:           false,
	reflect.Slice:         false,
	reflect.String:        true,
	reflect.Struct:        false,
	reflect.UnsafePointer: true,
}

func extractStructType(o interface{}) (reflect.Type, error) {
	return extractStructTypeImpl(reflect.TypeOf(o), reflect.ValueOf(o))
}

func extractStructTypeImpl(targetType reflect.Type, targetValue reflect.Value) (reflect.Type, error) {
	targetKind := targetType.Kind()

	if targetKind == reflect.Struct {
		return targetType, nil
	} else if targetKind == reflect.Interface {
		if targetValue.Kind() == reflect.Slice || targetValue.Kind() == reflect.Array {
			if targetValue.Len() == 0 {
				return nil, &dbError{msg: "Can't handle Interface"}
			}
			v := targetValue.Index(0).Elem()
			return extractStructTypeImpl(v.Type(), v)
		}
		return nil, &dbError{msg: "Can't handle Interface"}
	} else if nonStructElement[targetKind] {
		return nil, &dbError{msg: fmt.Sprintf("Element but not struct: %s", targetType.Kind())}
	}
	return extractStructTypeImpl(targetType.Elem(), targetValue)
}

func isAllSameType(o []interface{}) bool {
	if len(o) == 0 {
		return true
	}
	targetType := reflect.TypeOf(o[0])
	for i := range o {
		if reflect.TypeOf(o[i]).Name() != targetType.Name() {
			return false
		}
	}
	return true
}

// Select returns the list matching the query through argument bucket.
// Only slice is allowed to the argument `bucket`
func (client *DBClient) Select(bucket interface{}, query string, args ...interface{}) (bool, error) {
	if !client.IsOpen() {
		return false, &dbError{msg: "Database is not open yet"}
	}

	query = strings.TrimSpace(query)
	if !strings.HasPrefix(query, "where") {
		return false, &dbError{msg: "Query string must start with 'where'"}
	}

	t := reflect.TypeOf(bucket)
	if t.Kind() != reflect.Ptr {
		return false, &dbError{msg: "Argument 'bucket' must be a pointer to slice"}
	}
	if t.Elem().Kind() != reflect.Slice {
		return false, &dbError{msg: "Argument 'bucket' must be a slice"}
	}

	tableMap, err := client.dbmap.TableFor(t.Elem().Elem(), false)
	if err != nil {
		return false, err
	}

	query = "select * from " + tableMap.TableName + " " + query
	client.mutex.Lock()
	_, err = client.dbmap.Select(bucket, query, args...)
	client.mutex.Unlock()
	return err == nil, err
}

// Delete deletes by appending a query starting with 'where'
func (client *DBClient) Delete(typeIndicator interface{}, query string, args ...interface{}) (bool, error) {
	if !client.IsOpen() {
		return false, &dbError{msg: "Database is not open yet"}
	}

	query = strings.TrimSpace(query)
	if !strings.HasPrefix(query, "where") {
		return false, &dbError{msg: "Query string must start with 'where'"}
	}

	tableMap, err := client.dbmap.TableFor(reflect.TypeOf(typeIndicator), false)
	if err != nil {
		return false, err
	}

	query = "delete from " + tableMap.TableName + " " + query
	client.mutex.Lock()
	_, err = client.dbmap.Exec(query, args...)
	client.mutex.Unlock()
	return err == nil, err
}
