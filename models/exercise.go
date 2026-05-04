package models

// Exercise represents a coding exercise / problem
type Exercise struct {
	ID          string       `json:"id" firestore:"id"`
	Title       string       `json:"title" firestore:"title"`
	Difficulty  string       `json:"difficulty" firestore:"difficulty"` // Easy | Medium | Hard
	Description string       `json:"description" firestore:"description"`
	Requirement string       `json:"requirement" firestore:"requirement"`
	Example     string       `json:"example" firestore:"example"`
	Tips        []TipSection `json:"tips" firestore:"tips"`
}

// TipSection is a labeled block inside an exercise tip (e.g. "Pseudo Code")
type TipSection struct {
	Label   string `json:"label" firestore:"label"`
	Content string `json:"content" firestore:"content"`
}
