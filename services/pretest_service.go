package services

import (
	"algoplayground/config"
	"algoplayground/models"
	"context"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// GetPretestByAlgorithm fetches pretest questions from Firestore
// and transforms them into the frontend-compatible PretestResponse.
// Does NOT include correct answers in the response.
func GetPretestByAlgorithm(algorithm string) (*models.PretestResponse, error) {
	ctx := context.Background()

	query := config.Firestore.Collection("quizQuestions").
		Where("algorithm", "==", algorithm).
		Where("typeQuiz", "==", "pretest")

	iter := query.Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pretest questions: %v", err)
	}

	if len(docs) == 0 {
		return nil, nil
	}

	questions := make([]models.PretestQuestionDTO, 0, len(docs))

	for _, doc := range docs {
		var q models.QuizQuestion
		if err := doc.DataTo(&q); err != nil {
			fmt.Printf("Error mapping document %s: %v\n", doc.Ref.ID, err)
			continue
		}
		q.ID = doc.Ref.ID

		dto := models.PretestQuestionDTO{
			ID:            q.ID,
			Question:      q.Title,
			QuestionImage: q.QuestionImage,
		}

		// Parse choices (no correct answer included)
		if q.Question != nil {
			mapData, ok := q.Question.(map[string]interface{})
			if ok && q.Type == "multiple_choice" {
				var mc models.MultipleChoiceQuestion
				_ = mapstructure.Decode(mapData, &mc)

				choices := make([]models.PretestChoiceDTO, len(mc.Choices))
				for i, c := range mc.Choices {
					choices[i] = models.PretestChoiceDTO{
						ID:    fmt.Sprintf("%d", i),
						Label: string(rune('A' + i)),
						Text:  c.Label,
					}
				}
				dto.Choices = choices
			}
		}

		questions = append(questions, dto)
	}

	title := "Pretest of " + toTitleCase(algorithm)

	return &models.PretestResponse{
		ID:        "pretest-" + algorithm,
		Title:     title,
		Questions: questions,
	}, nil
}

// GradePretest grades user answers against correct answers from Firestore
// and saves the result to pretestResults collection.
func GradePretest(uid string, algorithm string, submission models.PretestSubmission) (*models.PretestGradingResult, error) {
	ctx := context.Background()

	// Fetch correct answers from Firestore
	query := config.Firestore.Collection("quizQuestions").
		Where("algorithm", "==", algorithm).
		Where("typeQuiz", "==", "pretest")

	iter := query.Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pretest questions for grading: %v", err)
	}

	// Build a map of questionId → correctChoiceId
	correctAnswers := make(map[string]string)
	for _, doc := range docs {
		var q models.QuizQuestion
		if err := doc.DataTo(&q); err != nil {
			continue
		}
		q.ID = doc.Ref.ID

		if q.Question != nil {
			mapData, ok := q.Question.(map[string]interface{})
			if ok && q.Type == "multiple_choice" {
				var mc models.MultipleChoiceQuestion
				_ = mapstructure.Decode(mapData, &mc)
				correctAnswers[q.ID] = fmt.Sprintf("%d", mc.CorrectChoiceIndex)
			}
		}
	}

	// Grade each answer
	score := 0
	results := make([]models.PretestQuestionResult, len(submission.Answers))

	for i, answer := range submission.Answers {
		correctId, exists := correctAnswers[answer.QuestionId]
		isCorrect := exists && answer.SelectedChoiceId == correctId

		if isCorrect {
			score++
		}

		results[i] = models.PretestQuestionResult{
			QuestionId: answer.QuestionId,
			IsCorrect:  isCorrect,
		}
	}

	gradingResult := &models.PretestGradingResult{
		Score:          score,
		TotalQuestions: len(submission.Answers),
		Results:        results,
	}

	// Save result to Firestore: pretestResults/{uid}_{algorithm}
	docID := fmt.Sprintf("%s_%s", uid, algorithm)
	_, err = config.Firestore.Collection("pretestResults").Doc(docID).Set(ctx, map[string]interface{}{
		"uid":            uid,
		"algorithm":      algorithm,
		"score":          score,
		"totalQuestions": len(submission.Answers),
		"answers":        submission.Answers,
		"results":        results,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save pretest result: %v", err)
	}

	return gradingResult, nil
}

// HasCompletedPretest checks if a user has already completed the pretest for an algorithm.
func HasCompletedPretest(uid string, algorithm string) (*models.PretestStatus, error) {
	ctx := context.Background()

	docID := fmt.Sprintf("%s_%s", uid, algorithm)
	doc, err := config.Firestore.Collection("pretestResults").Doc(docID).Get(ctx)

	if err != nil {
		// Document not found = not completed
		return &models.PretestStatus{Completed: false}, nil
	}

	data := doc.Data()
	score, _ := data["score"].(int64)
	total, _ := data["totalQuestions"].(int64)

	return &models.PretestStatus{
		Completed: true,
		Score:     int(score),
		Total:     int(total),
	}, nil
}

// toTitleCase converts a kebab-case slug to Title Case
func toTitleCase(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
