package database

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	client.Open()
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

// RegisterStruct registers struct types to gorp.DbMap
// Use this method as such:
//  	register := make([]interface{})
// 		register = append(register, Stock{})
//      register = append(register, User{})
// 		RegisterStruct(register)
func (client *DBClient) RegisterStruct(dummySlice []interface{}) {
	if !client.IsOpen() {
		return
	}

	for _, dummy := range dummySlice {
		fmt.Printf("%v", dummy)
		fmt.Printf("%v", client.dbmap)
		client.dbmap.AddTable(dummy)
	}
	err := client.dbmap.CreateTablesIfNotExists()
	if err != nil {
		logger.Error("Creating table failed: %s", err.Error())
	} else {
		fmt.Println("Created table")
	}
}
