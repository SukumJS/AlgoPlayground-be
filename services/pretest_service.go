package services

import (
	"algoplayground/config"
	"algoplayground/models"
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/mitchellh/mapstructure"
)

// progressDocID returns the Firestore document ID for a user's pretest progress
func progressDocID(uid, algorithm string) string {
	return fmt.Sprintf("%s_%s", uid, algorithm)
}

// GetPretestByAlgorithm fetches pretest questions.
// If the user has in-progress work, returns the SAME questions + saved answers.
// If not, fetches all questions and creates a new progress document.
func GetPretestByAlgorithm(uid string, algorithm string) (*models.PretestResponse, error) {
	ctx := context.Background()
	docID := progressDocID(uid, algorithm)

	// 1) Check for existing progress
	progressDoc, err := config.Firestore.Collection("pretestProgress").Doc(docID).Get(ctx)
	if err == nil {
		// Progress exists — fetch only the saved question IDs
		var progress models.PretestProgress
		if err := progressDoc.DataTo(&progress); err == nil && len(progress.QuestionIds) > 0 {
			questions, err := fetchQuestionsByIDs(ctx, progress.QuestionIds)
			if err != nil {
				return nil, err
			}

			return &models.PretestResponse{
				ID:           "pretest-" + algorithm,
				Title:        "Pretest of " + toTitleCase(algorithm),
				Questions:    questions,
				SavedAnswers: progress.Answers,
			}, nil
		}
	}

	// 2) No progress — fetch all questions for this algorithm
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
	questionIds := make([]string, 0, len(docs))

	for _, doc := range docs {
		var q models.QuizQuestion
		if err := doc.DataTo(&q); err != nil {
			fmt.Printf("Error mapping document %s: %v\n", doc.Ref.ID, err)
			continue
		}
		q.ID = doc.Ref.ID

		dto := transformQuestionToDTO(q)
		questions = append(questions, dto)
		questionIds = append(questionIds, q.ID)
	}

	// 3) Create progress document (empty answers)
	_, err = config.Firestore.Collection("pretestProgress").Doc(docID).Set(ctx, models.PretestProgress{
		UID:           uid,
		Algorithm:     algorithm,
		QuestionIds:   questionIds,
		Answers:       []models.PretestAnswer{},
		AnsweredCount: 0,
	})
	if err != nil {
		fmt.Printf("Warning: failed to create progress doc: %v\n", err)
	}

	return &models.PretestResponse{
		ID:        "pretest-" + algorithm,
		Title:     "Pretest of " + toTitleCase(algorithm),
		Questions: questions,
	}, nil
}

// fetchQuestionsByIDs fetches specific questions by their document IDs
func fetchQuestionsByIDs(ctx context.Context, ids []string) ([]models.PretestQuestionDTO, error) {
	questions := make([]models.PretestQuestionDTO, 0, len(ids))

	for _, id := range ids {
		doc, err := config.Firestore.Collection("quizQuestions").Doc(id).Get(ctx)
		if err != nil {
			fmt.Printf("Warning: question %s not found: %v\n", id, err)
			continue
		}

		var q models.QuizQuestion
		if err := doc.DataTo(&q); err != nil {
			fmt.Printf("Error mapping question %s: %v\n", id, err)
			continue
		}
		q.ID = doc.Ref.ID

		questions = append(questions, transformQuestionToDTO(q))
	}

	return questions, nil
}

// transformQuestionToDTO converts a QuizQuestion to a PretestQuestionDTO (no correct answer)
func transformQuestionToDTO(q models.QuizQuestion) models.PretestQuestionDTO {
	dto := models.PretestQuestionDTO{
		ID:            q.ID,
		Question:      q.Title,
		QuestionImage: q.QuestionImage,
	}

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

	return dto
}

// SavePretestProgress saves the user's partial answers to Firestore
func SavePretestProgress(uid string, algorithm string, answers []models.PretestAnswer) error {
	ctx := context.Background()
	docID := progressDocID(uid, algorithm)

	// Count how many questions have been answered
	answeredCount := 0
	for _, a := range answers {
		if a.SelectedChoiceId != "" {
			answeredCount++
		}
	}

	_, err := config.Firestore.Collection("pretestProgress").Doc(docID).Update(ctx, []firestore.Update{
		{Path: "answers", Value: answers},
		{Path: "answeredCount", Value: answeredCount},
	})

	if err != nil {
		return fmt.Errorf("failed to save pretest progress: %v", err)
	}

	return nil
}

// GradePretest grades user answers against correct answers from Firestore,
// saves the result, and deletes the progress document.
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

	// Save result to Firestore
	docID := progressDocID(uid, algorithm)
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

	// Delete progress document (pretest is now complete)
	_, _ = config.Firestore.Collection("pretestProgress").Doc(docID).Delete(ctx)

	return gradingResult, nil
}

// HasCompletedPretest checks the user's pretest state for an algorithm.
// Returns completed, in-progress, or not started.
func HasCompletedPretest(uid string, algorithm string) (*models.PretestStatus, error) {
	ctx := context.Background()
	docID := progressDocID(uid, algorithm)

	// Check completed first
	resultDoc, err := config.Firestore.Collection("pretestResults").Doc(docID).Get(ctx)
	if err == nil {
		data := resultDoc.Data()
		score, _ := data["score"].(int64)
		total, _ := data["totalQuestions"].(int64)

		return &models.PretestStatus{
			Completed: true,
			Score:     int(score),
			Total:     int(total),
		}, nil
	}

	// Check in-progress
	progressDoc, err := config.Firestore.Collection("pretestProgress").Doc(docID).Get(ctx)
	if err == nil {
		var progress models.PretestProgress
		if err := progressDoc.DataTo(&progress); err == nil {
			return &models.PretestStatus{
				InProgress:    true,
				AnsweredCount: progress.AnsweredCount,
				Total:         len(progress.QuestionIds),
			}, nil
		}
	}

	// Not started
	return &models.PretestStatus{}, nil
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

