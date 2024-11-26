package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type Session struct {
	ID       string
	UserName string
	UserID   string
}

var SessionKey = "o1VeB8ndLVc0NQXN"

func GenerateHexID() (string, error) {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func NewSession(name, userID string) *Session {
	ID, err := GenerateHexID()
	if err != nil {
		return nil
	}
	return &Session{ID: ID,
		UserName: name,
		UserID:   userID}
}

func CreateContextWithSession(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, SessionKey, sess)
}

func GetSessionFromContext(ctx context.Context) (*Session, error) {
	sess, ok := ctx.Value(SessionKey).(*Session)
	if !ok || sess == nil {
		return nil, ErrSessionNotFound
	}
	return sess, nil
}
