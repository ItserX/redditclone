package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"redditclone/pkg/session"
	"redditclone/pkg/user"
	"time"

	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap"
)

type UserHandler struct {
	Repo     user.UserRepo
	Sessions *session.SessionManager
	Logger   *zap.SugaredLogger
}

type LoginForm struct {
	Name     string `json:"username"`
	Password string `json:"password"`
}

var TokenSecret = []byte("nyEJB9GIy9aiwcRh")

func GenerateHexID() (string, error) {
	bytes := make([]byte, 12)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (handler *UserHandler) parseLoginForm(r *http.Request) (*LoginForm, error) {
	js, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	lf := &LoginForm{}
	err = json.Unmarshal(js, lf)
	if err != nil {
		return nil, err
	}
	return lf, nil
}

func (handler *UserHandler) createJWT(user *user.User, now int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": map[string]string{"username": user.Name, "id": user.ID},
		"iat":  now,
		"exp":  time.Now().Add(time.Hour * 12).Unix(),
	})

	tokenString, err := token.SignedString(TokenSecret)

	if err != nil {
		return "", err
	}
	return tokenString, nil
}

type LoginError struct {
	Location string `json:"location"`
	Param    string `json:"param"`
	Value    string `json:"value"`
	Msg      string `json:"msg"`
}

type LoginErrorResponse struct {
	Errors []LoginError `json:"errors"`
}

func (handler *UserHandler) sendRegisterError(w http.ResponseWriter, name string) {

	errorResponse := LoginErrorResponse{Errors: []LoginError{
		{Location: "body",
			Param: "username",
			Value: name,
			Msg:   user.ErrUserAlready.Error(),
		}}}

	jsonResp, err := json.Marshal(errorResponse)
	if err != nil {
		http.Error(w, ErrJSONMarshal.Error(), http.StatusInternalServerError)
		handler.Logger.Error(ErrJSONMarshal)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)

	_, errWrite := w.Write(jsonResp)
	if errWrite != nil {
		http.Error(w, errWrite.Error(), http.StatusInternalServerError)
		handler.Logger.Error(errWrite)
		return
	}
}

func (handler *UserHandler) sendLoginError(w http.ResponseWriter, errorMsg string) {
	resp, err := json.Marshal(map[string]string{"message": errorMsg})
	if err != nil {
		http.Error(w, ErrJSONMarshal.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_, errWrite := w.Write(resp)

	if errWrite != nil {
		http.Error(w, errWrite.Error(), http.StatusInternalServerError)
		handler.Logger.Error(errWrite)
		return
	}
}

func (handler *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	handler.Logger.Info("/register")
	_, err := handler.Sessions.CheckSession(r)

	if err == nil {
		err = handler.Sessions.DestroySession(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			handler.Logger.Error()
			return
		}
	}

	now := time.Now().Unix()

	lf, err := handler.parseLoginForm(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	userID, err := GenerateHexID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}

	currentUser := user.User{Name: lf.Name, Password: lf.Password, ID: userID}

	err = handler.Repo.AddUser(&currentUser)
	if err != nil {
		handler.sendRegisterError(w, currentUser.Name)
		handler.Logger.Error(err)
		return
	}

	token, err := handler.createJWT(&currentUser, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}
	resp, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		http.Error(w, ErrJSONMarshal.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}

	handler.Sessions.CreateSession(w, currentUser.Name, currentUser.ID)
	w.Header().Set("Content-Type", "application/json charset=utf-8")
	w.WriteHeader(http.StatusCreated)

	_, errWrite := w.Write(resp)
	if errWrite != nil {
		http.Error(w, errWrite.Error(), http.StatusInternalServerError)
		handler.Logger.Error(errWrite)
		return
	}
	handler.Logger.Infow("user regsitred",
		"ID", currentUser.ID,
		"Name", currentUser.Name)
}

func (handler *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	handler.Logger.Info("/login")
	_, err := handler.Sessions.CheckSession(r)

	if err == nil {
		err = handler.Sessions.DestroySession(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			handler.Logger.Error(err)
			return
		}
	}

	now := time.Now().Unix()

	lf, err := handler.parseLoginForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	switch handler.Repo.CheckUser(lf.Name, lf.Password) {
	case user.ErrInvalidPassword:
		handler.sendLoginError(w, user.ErrInvalidPassword.Error())
		handler.Logger.Error(user.ErrInvalidPassword)
		return
	case user.ErrUserNotExist:
		handler.sendLoginError(w, user.ErrUserNotExist.Error())
		handler.Logger.Error(user.ErrUserNotExist)
		return
	}

	currentUser, err := handler.Repo.GetUser(lf.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	token, err := handler.createJWT(currentUser, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}

	resp, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		http.Error(w, ErrJSONMarshal.Error(), http.StatusInternalServerError)
		handler.Logger.Error(ErrJSONMarshal)
		return
	}

	handler.Sessions.CreateSession(w, currentUser.Name, currentUser.ID)

	w.Header().Set("Content-Type", "application/json charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, errWrite := w.Write(resp)
	if errWrite != nil {
		http.Error(w, errWrite.Error(), http.StatusInternalServerError)
		handler.Logger.Error(errWrite)
		return
	}
	handler.Logger.Infow("user login success",
		"ID", currentUser.ID,
		"Name", currentUser.Name)
}
