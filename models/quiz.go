package models

// QuizQuestion is the top-level quiz question document
type QuizQuestion struct {
	ID            string      `json:"id" firestore:"id"`
	Type          string      `json:"type" firestore:"type"` // multiple_choice | ordering | fill_blank
	Algorithm     string      `json:"algorithm" firestore:"algorithm"`
	TypeQuiz      string      `json:"typeQuiz" firestore:"typeQuiz"` // pretest | posttest
	Title         string      `json:"title" firestore:"title"`
	QuestionImage string      `json:"questionImage" firestore:"questionImage"`
	Question      interface{} `json:"question" firestore:"question"` // polymorphic: MultipleChoiceQuestion | OrderingQuestion | FillQuestion
}

// --------------- Multiple Choice ---------------

// MultipleChoiceQuestion holds the choices and correct answer for a multiple-choice question
type MultipleChoiceQuestion struct {
	CorrectChoiceIndex int      `json:"correctChoiceIndex" firestore:"correctChoiceIndex"`
	Choices            []Choice `json:"choices" firestore:"choices"`
}

// Choice is a single option in a multiple-choice question
type Choice struct {
	Label string `json:"label" firestore:"label"`
}

// --------------- Fill Blank ---------------

// FillQuestion holds the correct answer for a fill-in-the-blank question
type FillQuestion struct {
	CorrectAnswer string `json:"correctAnswer" firestore:"correctAnswer"`
}

// --------------- Ordering ---------------

// OrderingQuestion holds items and the correct order for an ordering question
type OrderingQuestion struct {
	Items        []OrderingItem          `json:"items" firestore:"items"`
	CorrectOrder []OrderingCorrectAnswer `json:"correctOrder" firestore:"correctOrder"`
}

// OrderingItem is one draggable item in an ordering question
type OrderingItem struct {
	Label string `json:"label" firestore:"label"`
}

// OrderingCorrectAnswer defines the correct position for an ordering item
type OrderingCorrectAnswer struct {
	Label    string `json:"label" firestore:"label"`
	Position int    `json:"position" firestore:"position"`
}
