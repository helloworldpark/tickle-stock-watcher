package database

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"

	_ "github.com/go-sql-driver/mysql" // SQL Connection
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

// DBClient is a handle to database
type DBClient struct {
	ref           *commons.Ref
	mutex         sync.Mutex
	isInitialized bool
	db            *sql.DB
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
	ref.Retain()
	client := DBClient{ref: &ref}
	return &client
}

// Init initializes database connection
// Returns true if initialized, false else
func (client *DBClient) Init(credential DBCredential) bool {
	defer client.mutex.Unlock()

	client.mutex.Lock()
	if client.isInitialized {
		logger.Info("[DB] Already initialized")
		return true
	}

	openingQ := openingQuery(credential)
	db, err := sql.Open("mysql", openingQ)
	if err != nil {
		logger.Panic("%v", err)
		return false
	}
	logger.Info("[DB] Connected")
	client.db = db
	return true
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
	if client.ref.Count() == 2 {
		fmt.Println("Open")
	}
}

// Close closes connection to database only if there are no more needs for connection.
func (client *DBClient) Close() {
	defer client.mutex.Unlock()

	client.mutex.Lock()
	if client.ref.Count() <= 1 {
		return
	}
	client.ref.Release()
	if client.ref.Count() == 1 {
		fmt.Println("Close")
	}
}

// IsOpen returns true if database connection is open
func (client *DBClient) IsOpen() bool {
	defer client.mutex.Unlock()

	client.mutex.Lock()
	return client.ref.Count() > 1
}
