package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/SalvucciFacundo/templ-islands/examples/social/views"
)

// Comment is a reply to a post.
type Comment struct {
	ID     int    `json:"id"`
	PostID int    `json:"post_id"`
	Text   string `json:"text"`
}

// ChatMsg is one line in the simulated agent chat.
type ChatMsg struct {
	From string `json:"from"`
	Text string `json:"text"`
}

// Store is an in-memory post store. For a demo this is enough; a real app
// would use a database. The mutex keeps concurrent requests safe.
type Store struct {
	mu        sync.RWMutex
	posts     []views.Post
	next      int
	comments  map[int][]Comment
	nextComID int
	messages  []ChatMsg
}

func NewStore() *Store {
	s := &Store{next: 1, comments: make(map[int][]Comment), nextComID: 1}
	for i := 1; i <= 12; i++ {
		s.posts = append(s.posts, views.Post{
			ID:       i,
			Text:     fmt.Sprintf("Post de ejemplo #%d — renderizado en el servidor, isla en el cliente.", i),
			Likes:    i * 7,
			AuthorID: i,
		})
		s.comments[i] = []Comment{
			{ID: s.nextComID, PostID: i, Text: fmt.Sprintf("Comentario %d: me gusta este post.", s.nextComID)},
			{ID: s.nextComID + 1, PostID: i, Text: fmt.Sprintf("Comentario %d: ¿que linea es este canario?", s.nextComID+1)},
		}
		s.nextComID += 2
	}
	s.next = 13
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

// SearchPaged filtra por q (si no vacia) y pagina si page > 0.
// Con page <= 0 devuelve la lista filtrada completa.
func (s *Store) SearchPaged(q string, page, per int) []views.Post {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q = strings.ToLower(strings.TrimSpace(q))
	filtered := []views.Post{}
	for _, p := range s.posts {
		if q == "" || strings.Contains(strings.ToLower(p.Text), q) {
			filtered = append(filtered, p)
		}
	}
	if page <= 0 || per <= 0 {
		return filtered
	}
	start := (page - 1) * per
	if start >= len(filtered) {
		return []views.Post{}
	}
	end := start + per
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end]
}

// Comments returns the comments of a post.
func (s *Store) Comments(postID int) []Comment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Comment(nil), s.comments[postID]...)
}

// DeleteComment removes a comment and returns the updated list of its post.
func (s *Store) DeleteComment(id int) ([]Comment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for postID, list := range s.comments {
		for i, c := range list {
			if c.ID == id {
				s.comments[postID] = append(list[:i], list[i+1:]...)
				return append([]Comment(nil), s.comments[postID]...), true
			}
		}
	}
	return nil, false
}

// ChatMessages returns the chat history.
func (s *Store) ChatMessages() []ChatMsg {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]ChatMsg(nil), s.messages...)
}

// AddChatMessage appends a message to the chat history.
func (s *Store) AddChatMessage(from, text string) ChatMsg {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := ChatMsg{From: from, Text: text}
	s.messages = append(s.messages, m)
	return m
}
