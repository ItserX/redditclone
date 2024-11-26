package post

import (
	"errors"
	"math"
	"sync"
)

type PostsMemoryRepository struct {
	AllDataWithCategories map[string]map[string]*Post
	UserPostsData         map[string][]*Post
	AllData               map[string]*Post
	mu                    *sync.RWMutex
}

func NewPostMemoryRepository() *PostsMemoryRepository {
	allData := make(map[string]map[string]*Post)
	allData["music"] = make(map[string]*Post)
	allData["funny"] = make(map[string]*Post)
	allData["videos"] = make(map[string]*Post)
	allData["programming"] = make(map[string]*Post)
	allData["news"] = make(map[string]*Post)
	allData["fashion"] = make(map[string]*Post)

	repo := PostsMemoryRepository{
		AllDataWithCategories: allData,
		UserPostsData:         make(map[string][]*Post),
		AllData:               make(map[string]*Post),
		mu:                    &sync.RWMutex{},
	}

	return &repo
}

var ErrWrongCategory = errors.New("wrong category")
var ErrPostNotFound = errors.New("post not found")
var ErrCommentNotFound = errors.New("comment not found")
var ErrAccessDenied = errors.New("access denied")
var ErrUserNotFound = errors.New("user not found")

func (repo *PostsMemoryRepository) AddPost(p *Post) error {
	if _, ok := repo.AllDataWithCategories[p.Category]; !ok {
		return ErrWrongCategory
	}

	repo.mu.Lock()
	repo.AllData[p.ID] = p
	repo.AllDataWithCategories[p.Category][p.ID] = p
	repo.mu.Unlock()

	return nil
}

func (repo *PostsMemoryRepository) AddUserPost(userName string, p *Post) error {
	if _, ok := repo.UserPostsData[userName]; !ok {
		repo.mu.Lock()
		posts := make([]*Post, 0)
		repo.UserPostsData[userName] = posts
		repo.mu.Unlock()
	}
	repo.mu.Lock()
	repo.UserPostsData[userName] = append(repo.UserPostsData[userName], p)
	repo.mu.Unlock()
	return nil
}

func (repo *PostsMemoryRepository) GetPost(postID string) (*Post, error) {
	if post, ok := repo.AllData[postID]; !ok {
		return nil, ErrPostNotFound
	} else {
		return post, nil
	}
}

func (repo *PostsMemoryRepository) AddViews(post *Post) {
	repo.mu.Lock()
	post.Views += 1
	repo.mu.Unlock()
}

func (repo *PostsMemoryRepository) GetAllPosts() map[string]*Post {
	return repo.AllData
}

func (repo *PostsMemoryRepository) GetPostsWithCategory(category string) (map[string]*Post, error) {
	if posts, ok := repo.AllDataWithCategories[category]; !ok {
		return nil, ErrWrongCategory
	} else {
		return posts, nil
	}
}

func (repo *PostsMemoryRepository) AddCommentToPost(postID string, comment *Comment) error {
	if post, ok := repo.AllData[postID]; !ok {
		return ErrPostNotFound
	} else {
		repo.mu.Lock()
		post.Comments = append(post.Comments, comment)
		repo.mu.Unlock()
		return nil
	}
}

func (repo *PostsMemoryRepository) DeleteComment(post *Post, commentID string, userID string) error {
	flag := false
	index := -1
	for i, comm := range post.Comments {
		if comm.ID == commentID {
			flag = true
			index = i
			break
		}
	}

	if post.Comments[index].UserAuthor.ID != userID && index != -1 {
		return ErrAccessDenied
	}

	if flag {
		repo.mu.Lock()
		copy(post.Comments[index:], post.Comments[index+1:])
		post.Comments = post.Comments[:len(post.Comments)-1]
		repo.mu.Unlock()
		return nil
	} else {
		return ErrCommentNotFound
	}

}

func (repo *PostsMemoryRepository) issueScoreAndPercentage(p *Post) {
	downvoteScore := 0
	upvoteScore := 0
	p.Score = 0

	for _, v := range p.Votes {
		if v.Vote == UpvoteValue {
			upvoteScore += 1
		} else {
			downvoteScore += 1
		}
		p.Score += v.Vote
	}
	if p.Score == 0 {
		p.UpvotePercentage = 0
	} else {
		if downvoteScore != 0 {
			p.UpvotePercentage = int(math.Round(float64(upvoteScore) / float64(downvoteScore) * 100))
		} else {
			p.UpvotePercentage = 100
		}
	}
}

func (repo *PostsMemoryRepository) AddVote(p *Post, v *Vote, voteValue int) {
	flag := false
	index := -1
	for i, vote := range p.Votes {
		if v.UserID == vote.UserID {
			flag = true
			index = i
			break
		}
	}
	repo.mu.Lock()
	if flag {
		p.Votes[index].Vote = voteValue
	} else {
		p.Votes = append(p.Votes, v)
	}
	repo.issueScoreAndPercentage(p)
	repo.mu.Unlock()
}

func (repo *PostsMemoryRepository) DeleteVote(p *Post, userID string) error {
	flag := false
	index := -1
	for i, vote := range p.Votes {
		if vote.UserID == userID {
			flag = true
			index = i
			break
		}
	}
	if !flag {
		return ErrAccessDenied
	}

	repo.mu.Lock()
	copy(p.Votes[index:], p.Votes[index+1:])
	p.Votes = p.Votes[:len(p.Votes)-1]
	repo.issueScoreAndPercentage(p)
	repo.mu.Unlock()

	return nil
}

func (repo *PostsMemoryRepository) DeletePost(p *Post, userName, userID string) error {
	if userID != p.Author.ID {
		return ErrAccessDenied
	}
	repo.mu.Lock()
	delete(repo.AllData, p.ID)
	delete(repo.AllDataWithCategories[p.Category], p.ID)

	index := -1
	for i, post := range repo.UserPostsData[userName] {
		if post.ID == p.ID {
			index = i
			break
		}
	}
	copy(repo.UserPostsData[userName][index:], repo.UserPostsData[userName][index+1:])
	repo.UserPostsData[userName] = repo.UserPostsData[userName][:len(repo.UserPostsData[userName])-1]
	repo.mu.Unlock()
	return nil
}

func (repo *PostsMemoryRepository) GetUserPosts(userName string) ([]*Post, error) {
	if posts, ok := repo.UserPostsData[userName]; !ok {
		return nil, ErrUserNotFound
	} else {
		return posts, nil
	}

}
