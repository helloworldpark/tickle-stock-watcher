package structs

import "github.com/helloworldpark/tickle-stock-watcher/database"

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
