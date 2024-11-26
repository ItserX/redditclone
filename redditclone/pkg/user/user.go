package user

type User struct {
	Name     string
	Password string
	ID       string
}

type UserRepo interface {
	CheckUser(name, password string) error
	AddUser(user *User) error
	GetUser(name string) (*User, error)
}
