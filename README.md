# redditclone
Написал backend для копии reddit. Frontend взят отсюда https://github.com/SolarityVR/solarity-dream

1) POST /api/register - регистрация
2) POST /api/login - логин
3) GET /api/posts/ - список всех постов
4) POST /api/posts/ - добавление поста
5) GET /api/posts/{CATEGORY_NAME} - список постов конкретной категории
6) GET /api/post/{POST_ID} - детали поста с комментами
7) POST /api/post/{POST_ID} - добавление коммента
8) DELETE /api/post/{POST_ID}/{COMMENT_ID} - удаление коммента
9) GET /api/post/{POST_ID}/upvote - рейтинг поста вверх
10) GET /api/post/{POST_ID}/downvote - рейтинг поста вниз
11) GET /api/post/{POST_ID}/unvote - отмена голоса 
12) DELETE /api/post/{POST_ID} - удаление поста
13) GET /api/user/{USER_LOGIN} - получение всех постов конкретного пользователя

Запускиз папки redditclone: go run cmd/redditclone/main.go
