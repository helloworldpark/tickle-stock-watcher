package structs

import "github.com/helloworldpark/tickle-stock-watcher/database"
import "github.com/helloworldpark/tickle-stock-watcher/logger"

// User is a struct for describing users
type User struct {
	UserID    int64 // same as telegram id
	Superuser bool
}

// GetDBRegisterForm is just an implementation
func (s User) GetDBRegisterForm() database.DBRegisterForm {
	form := database.DBRegisterForm{
		BaseStruct:    User{},
		KeyColumns:    []string{"UserID"},
		AutoIncrement: false,
	}
	return form
}

// UserFromID finds user by user ID
func UserFromID(client *database.DBClient, token int64) (User, error) {
	var user []User
	_, err := client.Select(&user, "where UserID=?", token)
	if err != nil {
		logger.Error("[Structs] Error while selecting users: %s", err.Error())
		return User{}, err
	}
	if len(user) == 0 {
		return User{}, nil
	}
	return user[0], nil
}

// AllUsers returns all users
func AllUsers(client *database.DBClient) []User {
	var userList []User
	_, err := client.Select(&userList, "where true")
	if err != nil {
		logger.Error("[Structs] Error while selecting users: %s", err.Error())
	}
	return userList
}
