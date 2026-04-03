package models

// ── GET /posttests/:algorithm — response ────────────────────────

// PosttestResponse is the top-level response
type PosttestResponse struct {
	ID        string               `json:"id"`
	Title     string               `json:"title"`
	Questions []PosttestQuestionDTO `json:"questions"`
}

// PosttestQuestionDTO is one posttest question
type PosttestQuestionDTO struct {
	ID            string      `json:"id"`
	Type          string      `json:"type"` // multiple_choice | fill_blank | ordering
	Title         string      `json:"title"`
	Text          string      `json:"text"`
	QuestionImage string      `json:"questionImage,omitempty"`
	Question      interface{} `json:"question"` // type-specific payload
}

// PosttestMultipleChoiceDTO — with correct answer (client-side grading)
type PosttestMultipleChoiceDTO struct {
	MultipleChoice struct {
		Choices         []PosttestChoiceDTO `json:"choices"`
		CorrectChoiceId string              `json:"correctChoiceId"`
	} `json:"multipleChoice"`
}

// PosttestChoiceDTO is one answer choice
type PosttestChoiceDTO struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Text  string `json:"text"`
}

// PosttestFillBlankDTO — with correct answer
type PosttestFillBlankDTO struct {
	CorrectAnswer string `json:"correctAnswer"`
}

// PosttestOrderingDTO — with correct order
type PosttestOrderingDTO struct {
	Items        []PosttestOrderItemDTO `json:"items"`
	CorrectOrder []string               `json:"correctOrder"`
}

// PosttestOrderItemDTO is one draggable item
type PosttestOrderItemDTO struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
