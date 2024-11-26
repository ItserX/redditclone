package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	post "redditclone/pkg/posts"
	"redditclone/pkg/session"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

type PostHandler struct {
	Repo   post.PostRepo
	Logger *zap.SugaredLogger
}

type RequestForm struct {
	Category string `json:"category"`
	Text     string `json:"text"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	URL      string `json:"url"`
}

var ErrWrongCategory = errors.New("wrong category")
var ErrJSONMarshal = errors.New("json marshal error")
var ErrSessionNotFound = errors.New("session not found")
var ErrReadReqBody = errors.New("read request body error")
var ErrJSONUnmarshal = errors.New("json unmarshal error")

var HexIDSize = 12

func (handler *PostHandler) makeFormDate() string {
	date := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), time.Now().Hour(),
		time.Now().Minute(), time.Now().Second(), time.Now().Nanosecond(), time.UTC)

	formDate := date.Format(time.RFC3339Nano)
	return formDate
}

func (handler *PostHandler) generateHexID() (string, error) {
	bytes := make([]byte, HexIDSize)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (handler *PostHandler) SendPost(w http.ResponseWriter, currentPost post.Post) error {
	resp, err := json.Marshal(currentPost)
	if err != nil {
		http.Error(w, ErrJSONMarshal.Error(), http.StatusInternalServerError)
		return err
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_, errWrite := w.Write(resp)
	if errWrite != nil {
		http.Error(w, errWrite.Error(), http.StatusInternalServerError)
		return errWrite
	}
	return nil
}

func (handler *PostHandler) AddPost(w http.ResponseWriter, r *http.Request) {
	handler.Logger.Info("adding post")
	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		http.Error(w, ErrSessionNotFound.Error(), http.StatusUnauthorized)
		handler.Logger.Error(err)
		return
	}

	js, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, ErrReadReqBody.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}

	rf := &RequestForm{}

	err = json.Unmarshal(js, rf)
	if err != nil {
		http.Error(w, ErrJSONUnmarshal.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}

	postID, err := handler.generateHexID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}

	var currentPost = post.Post{}
	if rf.Type == "text" {
		currentPost = post.Post{
			Score:            1,
			Views:            0,
			Type:             rf.Type,
			Title:            rf.Title,
			Author:           post.Author{Username: sess.UserName, ID: sess.UserID},
			Category:         rf.Category,
			Text:             rf.Text,
			Votes:            make([]*post.Vote, 0),
			Comments:         make([]*post.Comment, 0),
			CreatedTime:      handler.makeFormDate(),
			UpvotePercentage: 100,
			ID:               postID,
		}
	} else if rf.Type == "link" {
		currentPost = post.Post{
			Score:            1,
			Views:            0,
			Type:             rf.Type,
			Title:            rf.Title,
			Author:           post.Author{Username: sess.UserName, ID: sess.UserID},
			Category:         rf.Category,
			URL:              rf.URL,
			Votes:            make([]*post.Vote, 0),
			Comments:         make([]*post.Comment, 0),
			CreatedTime:      handler.makeFormDate(),
			UpvotePercentage: 100,
			ID:               postID,
		}
	}

	currentPost.Votes = append(currentPost.Votes, &post.Vote{UserID: sess.UserID, Vote: post.UpvoteValue})

	err = handler.Repo.AddPost(&currentPost)
	if err != nil {
		http.Error(w, ErrWrongCategory.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	err = handler.Repo.AddUserPost(sess.UserName, &currentPost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	err = handler.SendPost(w, currentPost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}
}

func (handler *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	postID := strings.Replace(r.URL.Path, "/api/post/", "", 1)

	currentPost, err := handler.Repo.GetPost(postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}

	handler.Repo.AddViews(currentPost)

	resp, err := json.Marshal(currentPost)
	if err != nil {
		http.Error(w, ErrJSONMarshal.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, errWrite := w.Write(resp)
	if errWrite != nil {
		http.Error(w, errWrite.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}
	handler.Logger.Infow("post added",
		"postID", currentPost.ID)

}

func (handler *PostHandler) sortPostsAndSend(w http.ResponseWriter, currentPosts map[string]*post.Post) {

	posts := make([]*post.Post, 0)

	for _, val := range currentPosts {
		posts = append(posts, val)
	}

	sort.Slice(posts, func(i, j int) bool {
		return len(posts[i].Comments) < len(posts[j].Comments)
	})

	resp, err := json.Marshal(posts)
	if err != nil {
		http.Error(w, ErrJSONMarshal.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, errWrite := w.Write(resp)
	if errWrite != nil {
		http.Error(w, errWrite.Error(), http.StatusInternalServerError)
		handler.Logger.Error(errWrite)
		return
	}
}

func (handler *PostHandler) GetAllPosts(w http.ResponseWriter, r *http.Request) {

	currentPosts := handler.Repo.GetAllPosts()
	handler.sortPostsAndSend(w, currentPosts)

}

func (handler *PostHandler) GetPostsWithCategory(w http.ResponseWriter, r *http.Request) {

	category := strings.Replace(r.URL.Path, "/api/posts/", "", 1)

	currentPosts, err := handler.Repo.GetPostsWithCategory(category)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	handler.sortPostsAndSend(w, currentPosts)
}

type CommentRequest struct {
	Comment string `json:"comment"`
}

type CommentError struct {
	Location string `json:"location"`
	Param    string `json:"comment"`
	Msg      string `json:"msg"`
}

type ResponseCommentError struct {
	Errors []CommentError `json:"errors"`
}

func (handler *PostHandler) SendAddCommentError(w http.ResponseWriter) {
	resp := ResponseCommentError{
		Errors: []CommentError{
			{Location: "body",
				Param: "comment",
				Msg:   "is required"},
		},
	}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "json marshal error", http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_, errWrite := w.Write(js)
	if errWrite != nil {
		http.Error(w, errWrite.Error(), http.StatusInternalServerError)
		handler.Logger.Error(errWrite)
		return
	}

}

func (handler *PostHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	handler.Logger.Info("add comment")
	postID := strings.Replace(r.URL.Path, "/api/post/", "", 1)
	currentPost, err := handler.Repo.GetPost(postID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		http.Error(w, ErrSessionNotFound.Error(), http.StatusUnauthorized)
		handler.Logger.Error(err)
		return
	}

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}

	if len(reqBody) == 2 {
		handler.SendAddCommentError(w)
		return
	}
	cq := &CommentRequest{}

	err = json.Unmarshal(reqBody, cq)
	if err != nil {
		http.Error(w, ErrJSONUnmarshal.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}

	commentID, err := handler.generateHexID()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}
	currentComment := post.Comment{
		Body:        cq.Comment,
		UserAuthor:  post.Author{Username: sess.UserName, ID: sess.UserID},
		CreatedTime: handler.makeFormDate(),
		ID:          commentID,
	}

	err = handler.Repo.AddCommentToPost(currentPost.ID, &currentComment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}
	err = handler.SendPost(w, *currentPost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}
	handler.Logger.Infow("comment added",
		"commentID", commentID,
		"postID", postID)
}

func (handler *PostHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	handler.Logger.Info("delete comment")
	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		http.Error(w, ErrSessionNotFound.Error(), http.StatusUnauthorized)
		handler.Logger.Error(err)
		return
	}

	pathSegments := strings.Split(r.URL.Path, "/")[1:]

	postID := pathSegments[2]
	commentID := pathSegments[3]
	handler.Logger.Info(postID)
	currentPost, err := handler.Repo.GetPost(postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	err = handler.Repo.DeleteComment(currentPost, commentID, sess.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}
	err = handler.SendPost(w, *currentPost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}
	handler.Logger.Info("success")
}

func (handler *PostHandler) Upvote(w http.ResponseWriter, r *http.Request) {
	handler.Logger.Info("upvote")
	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		http.Error(w, ErrSessionNotFound.Error(), http.StatusUnauthorized)
		handler.Logger.Error(err)
		return
	}

	pathSegments := strings.Split(r.URL.Path, "/")[1:]

	postID := pathSegments[2]

	currentVote := &post.Vote{UserID: sess.UserID, Vote: post.UpvoteValue}

	currentPost, err := handler.Repo.GetPost(postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	handler.Repo.AddVote(currentPost, currentVote, post.UpvoteValue)
	err = handler.SendPost(w, *currentPost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}
	handler.Logger.Infow("success",
		"postID", postID)
}

func (handler *PostHandler) Downvote(w http.ResponseWriter, r *http.Request) {
	sess, err := session.GetSessionFromContext(r.Context())
	handler.Logger.Info("downvote")
	if err != nil {
		http.Error(w, ErrSessionNotFound.Error(), http.StatusUnauthorized)
		handler.Logger.Error(err)
		return
	}

	pathSegments := strings.Split(r.URL.Path, "/")[1:]

	postID := pathSegments[2]

	currentVote := &post.Vote{UserID: sess.UserID, Vote: post.DownvoteValue}

	currentPost, err := handler.Repo.GetPost(postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	handler.Repo.AddVote(currentPost, currentVote, post.DownvoteValue)
	err = handler.SendPost(w, *currentPost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}
	handler.Logger.Infow("success",
		"postID", postID)
}

func (handler *PostHandler) Unvote(w http.ResponseWriter, r *http.Request) {
	handler.Logger.Info("unvote")
	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		http.Error(w, ErrSessionNotFound.Error(), http.StatusUnauthorized)
		handler.Logger.Error(err)
		return
	}

	pathSegments := strings.Split(r.URL.Path, "/")[1:]

	postID := pathSegments[2]

	currentPost, err := handler.Repo.GetPost(postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	err = handler.Repo.DeleteVote(currentPost, sess.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}
	err = handler.SendPost(w, *currentPost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}
	handler.Logger.Infow("success",
		"postID", postID)
}

type DeletePostResponse struct {
	Message string `json:"message"`
}

func (handler *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	handler.Logger.Info("delete post")
	sess, err := session.GetSessionFromContext(r.Context())
	if err != nil {
		http.Error(w, ErrSessionNotFound.Error(), http.StatusUnauthorized)
		handler.Logger.Error(err)
		return
	}

	pathSegments := strings.Split(r.URL.Path, "/")[1:]

	postID := pathSegments[2]

	currentPost, err := handler.Repo.GetPost(postID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	err = handler.Repo.DeletePost(currentPost, sess.UserName, sess.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}
	resp, err := json.Marshal(DeletePostResponse{Message: "success"})
	if err != nil {
		http.Error(w, ErrJSONMarshal.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, errWrite := w.Write(resp)
	if errWrite != nil {
		http.Error(w, errWrite.Error(), http.StatusInternalServerError)
		handler.Logger.Error(errWrite)
		return
	}
	handler.Logger.Infow("success",
		"postID", postID)
}

func (handler *PostHandler) GetUserPosts(w http.ResponseWriter, r *http.Request) {
	handler.Logger.Info("get userPosts")
	pathSegments := strings.Split(r.URL.Path, "/")[1:]
	userName := pathSegments[2]

	posts, err := handler.Repo.GetUserPosts(userName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		handler.Logger.Error(err)
		return
	}

	resp, err := json.Marshal(posts)
	if err != nil {
		http.Error(w, ErrJSONMarshal.Error(), http.StatusInternalServerError)
		handler.Logger.Error(err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, errWrite := w.Write(resp)
	if errWrite != nil {
		http.Error(w, errWrite.Error(), http.StatusInternalServerError)
		handler.Logger.Error(errWrite)
		return
	}
	handler.Logger.Infow("success")
}
