package middleware

import (
	"net/http"
	"redditclone/pkg/session"
)

var noAuthUrls = map[string]string{

	"/api/login":      "POST",
	"/api/register":   "POST",
	"/api/posts/":     "GET",
	"/api/post/{ID}":  "GET",
	"/api/user/{ID}":  "GET",
	"/api/posts/{ID}": "GET",
}

func Auth(sm *session.SessionManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if val, ok := noAuthUrls[r.URL.Path]; ok && val == r.Method {
			next.ServeHTTP(w, r)
			return
		}

		sess, _ := sm.CheckSession(r) //nolint:errcheck

		ctx := session.CreateContextWithSession(r.Context(), sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
