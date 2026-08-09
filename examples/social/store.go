package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/SalvucciFacundo/templ-islands/examples/social/views"
)

// Store is an in-memory post store. For a demo this is enough; a real app
// would use a database. The mutex keeps concurrent requests safe.
type Store struct {
	mu    sync.RWMutex
	posts []views.Post
	next  int
}

func NewStore() *Store {
	s := &Store{next: 1}
	for i := 1; i <= 5; i++ {
		s.posts = append(s.posts, views.Post{
			ID:       i,
			Text:     fmt.Sprintf("Post de ejemplo #%d — renderizado en el servidor, isla en el cliente.", i),
			Likes:    i * 7,
			AuthorID: i,
		})
	}
	s.next = 6
	return s
}

func (s *Store) Posts() []views.Post {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]views.Post, len(s.posts))
	copy(out, s.posts)
	return out
}

func (s *Store) Get(id int) (views.Post, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.posts {
		if p.ID == id {
			return p, true
		}
	}
	return views.Post{}, false
}

// Like flips the like state and returns the updated post.
func (s *Store) Like(id int) (views.Post, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.posts {
		if s.posts[i].ID == id {
			p := &s.posts[i]
			if p.Liked {
				p.Liked = false
				p.Likes--
			} else {
				p.Liked = true
				p.Likes++
			}
			return *p, true
		}
	}
	return views.Post{}, false
}

// Follow flips the follow state for an author and returns one updated post.
func (s *Store) Follow(authorID int) (views.Post, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.posts {
		if s.posts[i].AuthorID == authorID {
			p := &s.posts[i]
			p.Following = !p.Following
			return *p, true
		}
	}
	return views.Post{}, false
}

// Create appends a new post and returns it.
func (s *Store) Create(text string) views.Post {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := views.Post{ID: s.next, Text: text, AuthorID: s.next}
	s.next++
	s.posts = append(s.posts, p)
	return p
}

// Search returns posts whose text contains q (case-insensitive).
func (s *Store) Search(q string) []views.Post {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q = strings.ToLower(strings.TrimSpace(q))
	out := []views.Post{}
	for _, p := range s.posts {
		if q == "" || strings.Contains(strings.ToLower(p.Text), q) {
			out = append(out, p)
		}
	}
	return out
}
