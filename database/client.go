package database

import "github.com/helloworldpark/tickle-stock-watcher/commons"
import "fmt"

// DBClient is a handle to database
type DBClient struct {
	ref *commons.Ref
}

// Create creates DBClient struct
func Create() *DBClient {
	ref := commons.Ref{}
	ref.Retain()
	client := DBClient{ref: &ref}
	return &client
}

// Open may start a connection to database if there are no active connections.
// Calling this method many times does not have any effect to overconnection.
func (client DBClient) Open() {
	client.ref.Retain()
	if client.ref.Count() == 2 {
		fmt.Println("Open")
	}
}

// Close closes connection to database only if there are no more needs for connection.
func (client DBClient) Close() {
	if client.ref.Count() <= 1 {
		return
	}
	client.ref.Release()
	if client.ref.Count() == 1 {
		fmt.Println("Close")
	}
}
