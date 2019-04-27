package structs

import "github.com/helloworldpark/tickle-stock-watcher/database"
import "github.com/helloworldpark/tickle-stock-watcher/logger"

// User is a struct for describing users
type User struct {
	UserID        int
	NameLine      string
	NameTelegram  string
	NameKakao     string
	TokenLine     string
	TokenTelegram string
	TokenKakao    string
	Superuser     bool
}

// GetDBRegisterForm is just an implementation
func (s User) GetDBRegisterForm() database.DBRegisterForm {
	form := database.DBRegisterForm{
		BaseStruct:    User{},
		KeyColumns:    []string{"UserID"},
		AutoIncrement: true,
	}
	return form
}

// AllUsers returns all users
func AllUsers(client *database.DBClient) []User {
	var userList []User
	_, err := client.Select(&userList, "where true")
	if err != nil {
		logger.Panic("Error while selecting users: %s", err.Error())
	}
	return userList
}
