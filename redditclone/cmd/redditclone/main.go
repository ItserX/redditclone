package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"redditclone/middleware"
	"redditclone/pkg/handlers"
	post "redditclone/pkg/posts"
	"redditclone/pkg/session"
	"redditclone/pkg/user"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func AddHandleFuncs(r *mux.Router, f handlers.UserHandler, p handlers.PostHandler) {
	r.HandleFunc("/api/register", f.Register).Methods("POST")
	r.HandleFunc("/api/login", f.Login).Methods("POST")
	r.HandleFunc("/api/posts", p.AddPost).Methods("POST")
	r.HandleFunc("/api/posts/", p.GetAllPosts).Methods("GET")
	r.HandleFunc("/api/post/{ID}", p.GetPost).Methods("GET")
	r.HandleFunc("/api/posts/{ID}", p.GetPostsWithCategory).Methods("GET")
	r.HandleFunc("/api/post/{ID}", p.AddComment).Methods("POST")
	r.HandleFunc("/api/post/{ID}/{ID}", p.DeleteComment).Methods("DELETE")
	r.HandleFunc("/api/post/{ID}/upvote", p.Upvote).Methods("GET")
	r.HandleFunc("/api/post/{ID}/downvote", p.Downvote).Methods("GET")
	r.HandleFunc("/api/post/{ID}/unvote", p.Unvote).Methods("GET")
	r.HandleFunc("/api/post/{ID}", p.DeletePost).Methods("DELETE")
	r.HandleFunc("/api/user/{ID}", p.GetUserPosts).Methods("GET")
}

func main() {
	r := mux.NewRouter()

	rootDir, err := os.Getwd()
	fmt.Println(rootDir)
	if err != nil {
		fmt.Println("get root directory error")
	}

	staticHTMLPath := filepath.Join(rootDir, "../", "../", "static", "html", "index.html")
	staticDirPath := filepath.Join(rootDir, "../", "../", "static")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, staticHTMLPath)
	})

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticDirPath))))

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Println("new logger error")
	}
	lg := logger.Sugar()

	sm := session.NewSessionsManager()
	f := handlers.UserHandler{Repo: user.NewUserMemRep(), Sessions: sm, Logger: lg}
	p := handlers.PostHandler{Repo: post.NewPostMemoryRepository(), Logger: lg}
	AddHandleFuncs(r, f, p)

	mux := middleware.Auth(sm, r)
	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Println("ListenAndServe error")
	}
}
