package models

// ── GET /pretests/:algorithm — response ─────────────────────────

// PretestResponse is the top-level response for GET /pretests/:algorithm
type PretestResponse struct {
	ID           string              `json:"id"`
	Title        string              `json:"title"`
	Questions    []PretestQuestionDTO `json:"questions"`
	SavedAnswers []PretestAnswer     `json:"savedAnswers,omitempty"`
}

// PretestQuestionDTO is one question (NO correct answer sent to frontend)
type PretestQuestionDTO struct {
	ID            string             `json:"id"`
	Question      string             `json:"question"`
	QuestionImage string             `json:"questionImage,omitempty"`
	Choices       []PretestChoiceDTO `json:"choices"`
}

// PretestChoiceDTO is one answer choice
type PretestChoiceDTO struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Text  string `json:"text"`
}

// ── POST /pretests/:algorithm/submit — request ──────────────────

// PretestSubmission is the request body when submitting answers
type PretestSubmission struct {
	Answers []PretestAnswer `json:"answers"`
}

// PretestAnswer is one user answer
type PretestAnswer struct {
	QuestionId       string `json:"questionId" firestore:"questionId"`
	SelectedChoiceId string `json:"selectedChoiceId" firestore:"selectedChoiceId"`
}

// ── POST /pretests/:algorithm/submit — response ─────────────────

// PretestGradingResult is the grading response (does NOT reveal correct answers)
type PretestGradingResult struct {
	Score          int                     `json:"score"`
	TotalQuestions int                     `json:"totalQuestions"`
	Results        []PretestQuestionResult `json:"results"`
}

// PretestQuestionResult tells if a single question was answered correctly
type PretestQuestionResult struct {
	QuestionId string `json:"questionId"`
	IsCorrect  bool   `json:"isCorrect"`
}

// ── GET /pretests/:algorithm/status — response ──────────────────

// PretestStatus tells the user's pretest state for an algorithm
type PretestStatus struct {
	Completed     bool `json:"completed"`
	InProgress    bool `json:"inProgress"`
	Score         int  `json:"score,omitempty"`
	Total         int  `json:"total,omitempty"`
	AnsweredCount int  `json:"answeredCount,omitempty"`
}

// ── PUT /pretests/:algorithm/progress — request ─────────────────

// PretestProgressRequest is the request body for saving progress
type PretestProgressRequest struct {
	Answers []PretestAnswer `json:"answers"`
}

// ── Firestore document: pretestProgress/{uid}_{algorithm} ───────

// PretestProgress is stored in Firestore to track in-progress pretests
type PretestProgress struct {
	UID           string          `firestore:"uid"`
	Algorithm     string          `firestore:"algorithm"`
	QuestionIds   []string        `firestore:"questionIds"`
	Answers       []PretestAnswer `firestore:"answers"`
	AnsweredCount int             `firestore:"answeredCount"`
}
