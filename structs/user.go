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

// UserFromToken finds user by token
func UserFromToken(client *database.DBClient, token, messenger string) (User, error) {
	var user []User
	var query string
	switch messenger {
	case "Telegram":
		query = "where TokenTelegram=?"
	case "Line":
		query = "where TokenLine=?"
	case "Kakao":
		query = "where TokenKakao=?"
	default:
		logger.Panic("[Structs] Should implement this case in User: %s", messenger)
	}
	_, err := client.Select(&user, query, token)
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
