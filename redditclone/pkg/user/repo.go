package user

import (
	"errors"
	"sync"
)

type UserMemoryRepository struct {
	data map[string]*User
	mu   sync.RWMutex
}

var ErrUserAlready = errors.New("already exist")
var ErrUserNotExist = errors.New("user not found")
var ErrInvalidPassword = errors.New("invalid password")

func NewUserMemRep() *UserMemoryRepository {
	return &UserMemoryRepository{data: make(map[string]*User)}
}

func (repo *UserMemoryRepository) CheckUser(name, password string) error {
	if _, ok := repo.data[name]; !ok {
		return ErrUserNotExist
	}

	if val, ok := repo.data[name]; ok && val.Password != password {
		return ErrInvalidPassword
	}

	return nil

}

func (repo *UserMemoryRepository) AddUser(user *User) error {
	if repo.CheckUser(user.Name, user.Password) != nil {

		repo.mu.Lock()
		repo.data[user.Name] = user
		repo.mu.Unlock()

		return nil
	}

	return ErrUserAlready

}

func (repo *UserMemoryRepository) GetUser(name string) (*User, error) {
	if val, ok := repo.data[name]; ok {
		return val, nil
	}
	return nil, ErrUserNotExist
}
