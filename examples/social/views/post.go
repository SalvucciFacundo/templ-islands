package views

// Post is a single item in the social feed. The JSON tags define the data
// contract shared with the client renderer (static/post-list.js).
type Post struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	Image     string `json:"image"`
	Likes     int    `json:"likes"`
	Liked     bool   `json:"liked"`
	AuthorID  int    `json:"author_id"`
	Following bool   `json:"following"`
}
