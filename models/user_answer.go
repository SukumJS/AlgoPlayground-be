package models

import (
	"time"
)

// UserAnswer is the top-level answer document submitted by a user
type UserAnswer struct {
	ID             string      `json:"id" firestore:"id"`
	UserID         string      `json:"userId" firestore:"userId"`
	QuizQuestionID string      `json:"quizQuestionId" firestore:"quizQuestionId"`
	Type           string      `json:"type" firestore:"type"` // multiple_choice | ordering | fill_blank
	Answer         interface{} `json:"answer" firestore:"answer"`
	CreatedAt      time.Time   `json:"createdAt" firestore:"createdAt"`
}

// --------------- Multiple Choice Answer ---------------

// UserAnswerMultipleChoice stores the user's selected choice(s)
type UserAnswerMultipleChoice struct {
	SelectedChoice int  `json:"selectedChoice" firestore:"selectedChoice"` // Store INDEX
	IsCorrect      bool `json:"isCorrect" firestore:"isCorrect"`
}

// --------------- Ordering Answer ---------------

// UserAnswerOrdering stores the user's ordering answer
type UserAnswerOrdering struct {
	// Store each item's label with its position.
	//
	// Example:
	// Correct order: [1,2,3]
	// Stored as:
	// [
	//   {Label: "item1", Position: 1},
	//   {Label: "item2", Position: 2},
	//   {Label: "item3", Position: 3}
	// ]
	//
	// If the user answers: [1,3,2]
	// Stored as:
	// [
	//   {Label: "item1", Position: 1},
	//   {Label: "item3", Position: 3},
	//   {Label: "item2", Position: 2}
	// ]
	//
	// Each object represents an item and the position where it was placed.
	Answer    []OrderingCorrectAnswer `json:"answer" firestore:"answer"`
	IsCorrect []bool                  `json:"isCorrect" firestore:"isCorrect"`
}

// --------------- Fill Blank Answer ---------------

// UserAnswerFillBlank stores the user's fill-in-the-blank answer
type UserAnswerFillBlank struct {
	Answer    string `json:"answer" firestore:"answer"`
	IsCorrect bool   `json:"isCorrect" firestore:"isCorrect"`
}
