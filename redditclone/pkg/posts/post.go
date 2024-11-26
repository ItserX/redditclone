package post

type Post struct {
	Score            int        `json:"score"`
	Views            int        `json:"views"`
	Type             string     `json:"type"`
	Title            string     `json:"title"`
	Category         string     `json:"category"`
	Text             string     `json:"text,omitempty"`
	Author           Author     `json:"author"`
	Votes            []*Vote    `json:"votes"`
	Comments         []*Comment `json:"comments"`
	UpvotePercentage int        `json:"upvotePercentage"`
	ID               string     `json:"id"`
	CreatedTime      string     `json:"created"`
	URL              string     `json:"url,omitempty"`
}

type Author struct {
	Username string `json:"username"`
	ID       string `json:"id"`
}

type Vote struct {
	UserID string `json:"user"`
	Vote   int    `json:"vote"`
}

type Comment struct {
	Body        string `json:"body"`
	UserAuthor  Author `json:"author"`
	CreatedTime string `json:"created"`
	ID          string `json:"id"`
}

const (
	UpvoteValue   = 1
	DownvoteValue = -1
)

type PostRepo interface {
	AddUserPost(userName string, p *Post) error
	GetPost(postID string) (*Post, error)
	AddViews(post *Post)
	GetPostsWithCategory(category string) (map[string]*Post, error)
	AddCommentToPost(postID string, comment *Comment) error
	DeleteComment(post *Post, commentID string, userID string) error
	issueScoreAndPercentage(p *Post)
	AddVote(p *Post, v *Vote, voteValue int)
	DeleteVote(p *Post, userID string) error
	DeletePost(p *Post, userName, userID string) error
	GetUserPosts(userName string) ([]*Post, error)
	AddPost(p *Post) error
	GetAllPosts() map[string]*Post
}
