package models

// ── GET /pretests/:algorithm — response ─────────────────────────

// PretestResponse is the top-level response for GET /pretests/:algorithm
type PretestResponse struct {
	ID        string              `json:"id"`
	Title     string              `json:"title"`
	Questions []PretestQuestionDTO `json:"questions"`
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
	QuestionId       string `json:"questionId"`
	SelectedChoiceId string `json:"selectedChoiceId"`
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

// PretestStatus tells if the user has completed the pretest
type PretestStatus struct {
	Completed bool `json:"completed"`
	Score     int  `json:"score,omitempty"`
	Total     int  `json:"total,omitempty"`
}
