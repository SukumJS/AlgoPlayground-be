package models

import "time"

// ── GET /posttests/:algorithm — response (NO correct answers) ───

// PosttestResponse is the top-level response
type PosttestResponse struct {
	ID           string               `json:"id"`
	Title        string               `json:"title"`
	Questions    []PosttestQuestionDTO `json:"questions"`
	SavedAnswers []PosttestAnswerDTO   `json:"savedAnswers,omitempty"`
}

// PosttestQuestionDTO is one posttest question (no correct answer)
type PosttestQuestionDTO struct {
	ID            string      `json:"id"`
	Type          string      `json:"type"` // multiple_choice | fill_blank | ordering
	Title         string      `json:"title"`
	Text          string      `json:"text"`
	QuestionImage string      `json:"questionImage,omitempty"`
	Question      interface{} `json:"question"` // type-specific payload
}

// PosttestMultipleChoiceDTO — choices only, NO correctChoiceId
type PosttestMultipleChoiceDTO struct {
	MultipleChoice struct {
		Choices []PosttestChoiceDTO `json:"choices"`
	} `json:"multipleChoice"`
}

// PosttestChoiceDTO is one answer choice
type PosttestChoiceDTO struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Text  string `json:"text"`
}

// PosttestFillBlankDTO — empty, NO correctAnswer
type PosttestFillBlankDTO struct{}

// PosttestOrderingDTO — items only, NO correctOrder
type PosttestOrderingDTO struct {
	Items      []PosttestOrderItemDTO `json:"items"`
	CanvasData *CanvasData            `json:"canvasData,omitempty"`
}

// PosttestOrderItemDTO is one draggable item
type PosttestOrderItemDTO struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ── POST /posttests/:algorithm/submit — request ─────────────────

// PosttestSubmission is the request body when submitting answers
type PosttestSubmission struct {
	Answers []PosttestAnswerDTO `json:"answers"`
}

// PosttestAnswerDTO is one user answer (supports all 3 types)
type PosttestAnswerDTO struct {
	QuestionId       string   `json:"questionId" firestore:"questionId"`
	Type             string   `json:"type" firestore:"type"`
	SelectedChoiceId string   `json:"selectedChoiceId,omitempty" firestore:"selectedChoiceId,omitempty"`
	FilledAnswer     string   `json:"filledAnswer,omitempty" firestore:"filledAnswer,omitempty"`
	OrderedItems     []string `json:"orderedItems,omitempty" firestore:"orderedItems,omitempty"`
}

// ── POST /posttests/:algorithm/submit — response ────────────────

// PosttestGradingResult is the grading response (includes correct answers for result page)
type PosttestGradingResult struct {
	Score          int                       `json:"score"`
	TotalQuestions int                       `json:"totalQuestions"`
	Results        []PosttestQuestionResult  `json:"results"`
}

// PosttestQuestionResult tells correctness + correct answer per question
type PosttestQuestionResult struct {
	QuestionId       string   `json:"questionId"`
	Type             string   `json:"type"`
	IsCorrect        bool     `json:"isCorrect"`
	CorrectChoiceId  string   `json:"correctChoiceId,omitempty"`  // multiple_choice
	CorrectAnswer    string   `json:"correctAnswer,omitempty"`    // fill_blank
	CorrectOrder     []string `json:"correctOrder,omitempty"`     // ordering
}

// ── GET /posttests/:algorithm/status — response ─────────────────

// PosttestStatus tells the user's posttest state
type PosttestStatus struct {
	Algorithm       string     `json:"algorithm"`
	Completed       bool       `json:"completed"`
	InProgress      bool       `json:"inProgress"`
	Score           *int       `json:"score"`
	Total           *int       `json:"total"`
	AnsweredCount   *int       `json:"answeredCount"`
	ReminderShown   bool       `json:"reminderShown"`
	ReminderShownAt *time.Time `json:"reminderShownAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// PosttestReminderState is the response payload for reminder-related updates.
type PosttestReminderState struct {
	Algorithm       string     `json:"algorithm"`
	ReminderShown   bool       `json:"reminderShown"`
	ReminderShownAt *time.Time `json:"reminderShownAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// PosttestReminderSeenRequest is the request body for marking reminder as seen.
type PosttestReminderSeenRequest struct {
	Seen   bool   `json:"seen"`
	Source string `json:"source,omitempty"`
}

// PosttestReminderResetRequest is the request body for resetting reminder state.
type PosttestReminderResetRequest struct {
	Reset bool `json:"reset"`
}

// ── PUT /posttests/:algorithm/progress — request ────────────────

// PosttestProgressRequest is the request body for saving progress
type PosttestProgressRequest struct {
	Answers []PosttestAnswerDTO `json:"answers"`
}

// ── Firestore: posttestProgress/{uid}_{algorithm} ───────────────

// PosttestProgress is stored in Firestore
type PosttestProgress struct {
	UID           string              `firestore:"uid"`
	Algorithm     string              `firestore:"algorithm"`
	QuestionIds   []string            `firestore:"questionIds"`
	Answers       []PosttestAnswerDTO `firestore:"answers"`
	AnsweredCount int                 `firestore:"answeredCount"`
}

// PosttestReminderRecord is stored in Firestore per user+algorithm.
type PosttestReminderRecord struct {
	UID             string     `firestore:"uid"`
	Algorithm       string     `firestore:"algorithm"`
	ReminderShown   bool       `firestore:"reminderShown"`
	ReminderShownAt *time.Time `firestore:"reminderShownAt"`
	Source          string     `firestore:"source,omitempty"`
	UpdatedAt       time.Time  `firestore:"updatedAt"`
}
