package services

import (
	"algoplayground/config"
	"algoplayground/models"
	"context"
	"fmt"
	"math/rand"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/mitchellh/mapstructure"
)

// postDocID returns Firestore doc ID for a user+algorithm pair
func postDocID(uid, algorithm string) string {
	return fmt.Sprintf("%s_%s", uid, algorithm)
}

// ── GET /posttests/:algorithm ───────────────────────────────────

// GetPosttestByAlgorithm returns questions, resuming progress if it exists.
func GetPosttestByAlgorithm(uid string, algorithm string) (*models.PosttestResponse, error) {
	ctx := context.Background()
	docID := postDocID(uid, algorithm)

	// 1) Check for existing progress
	progressDoc, err := config.Firestore.Collection("posttestProgress").Doc(docID).Get(ctx)
	if err == nil {
		var progress models.PosttestProgress
		if err := progressDoc.DataTo(&progress); err == nil && len(progress.QuestionIds) > 0 {
			questions, err := fetchPosttestQuestionsByIDs(ctx, progress.QuestionIds)
			if err != nil {
				return nil, err
			}

			return &models.PosttestResponse{
				ID:           "posttest-" + algorithm,
				Title:        "Post Test Of " + posttestTitleCase(algorithm),
				Questions:    questions,
				SavedAnswers: progress.Answers,
			}, nil
		}
	}

	// 2) No progress — fetch all questions and randomly select 5
	query := config.Firestore.Collection("quizQuestions").
		Where("algorithm", "==", algorithm).
		Where("typeQuiz", "==", "posttest")

	iter := query.Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posttest questions: %v", err)
	}

	if len(docs) == 0 {
		return nil, nil
	}

	allQuestions := make([]parsedPosttestQuestion, 0, len(docs))
	for _, doc := range docs {
		var q models.QuizQuestion
		if err := doc.DataTo(&q); err != nil {
			fmt.Printf("Error mapping posttest document %s: %v\n", doc.Ref.ID, err)
			continue
		}
		q.ID = doc.Ref.ID
		allQuestions = append(allQuestions, parsedPosttestQuestion{quiz: q})
	}

	selected := selectPosttestQuestions(allQuestions, 5)

	dtos := make([]models.PosttestQuestionDTO, 0, len(selected))
	questionIds := make([]string, 0, len(selected))
	for _, pq := range selected {
		dto := transformPosttestToDTO(pq.quiz)
		if dto != nil {
			dtos = append(dtos, *dto)
			questionIds = append(questionIds, pq.quiz.ID)
		}
	}

	// 3) Create progress document
	_, err = config.Firestore.Collection("posttestProgress").Doc(docID).Set(ctx, models.PosttestProgress{
		UID:           uid,
		Algorithm:     algorithm,
		QuestionIds:   questionIds,
		Answers:       []models.PosttestAnswerDTO{},
		AnsweredCount: 0,
	})
	if err != nil {
		fmt.Printf("Warning: failed to create posttest progress doc: %v\n", err)
	}

	return &models.PosttestResponse{
		ID:        "posttest-" + algorithm,
		Title:     "Post Test Of " + posttestTitleCase(algorithm),
		Questions: dtos,
	}, nil
}

// ── Save progress ───────────────────────────────────────────────

func SavePosttestProgress(uid string, algorithm string, answers []models.PosttestAnswerDTO) error {
	ctx := context.Background()
	docID := postDocID(uid, algorithm)

	answeredCount := 0
	for _, a := range answers {
		if a.SelectedChoiceId != "" || a.FilledAnswer != "" || len(a.OrderedItems) > 0 {
			answeredCount++
		}
	}

	// แก้ไขจาก Update เป็น Set พร้อม MergeAll
	_, err := config.Firestore.Collection("posttestProgress").Doc(docID).Set(ctx, map[string]interface{}{
		"answers":       answers,
		"answeredCount": answeredCount,
	}, firestore.MergeAll) // MergeAll จะช่วยให้อัปเดตเฉพาะฟิลด์โดยไม่ทับข้อมูลอื่นที่มีอยู่

	if err != nil {
		return fmt.Errorf("failed to save posttest progress: %v", err)
	}

	return nil
}

// ── Grade posttest ──────────────────────────────────────────────

func GradePosttest(uid string, algorithm string, submission models.PosttestSubmission) (*models.PosttestGradingResult, error) {
	ctx := context.Background()

	// Build answer map by questionId
	answerMap := make(map[string]models.PosttestAnswerDTO)
	for _, a := range submission.Answers {
		answerMap[a.QuestionId] = a
	}

	// Get the question IDs from progress (to know which questions were assigned)
	docID := postDocID(uid, algorithm)
	progressDoc, err := config.Firestore.Collection("posttestProgress").Doc(docID).Get(ctx)
	var questionIDs []string
	if err == nil {
		var progress models.PosttestProgress
		if err := progressDoc.DataTo(&progress); err == nil {
			questionIDs = progress.QuestionIds
		}
	}

	// If no progress, collect IDs from submitted answers
	if len(questionIDs) == 0 {
		for _, a := range submission.Answers {
			questionIDs = append(questionIDs, a.QuestionId)
		}
	}

	// Fetch the actual questions from Firestore to get correct answers
	score := 0
	results := make([]models.PosttestQuestionResult, 0, len(questionIDs))

	for _, qID := range questionIDs {
		doc, err := config.Firestore.Collection("quizQuestions").Doc(qID).Get(ctx)
		if err != nil {
			continue
		}

		var q models.QuizQuestion
		if err := doc.DataTo(&q); err != nil {
			continue
		}
		q.ID = doc.Ref.ID

		answer, hasAnswer := answerMap[qID]
		result := gradeOnePosttestQuestion(q, answer, hasAnswer)
		if result.IsCorrect {
			score++
		}
		results = append(results, result)
	}

	gradingResult := &models.PosttestGradingResult{
		Score:          score,
		TotalQuestions: len(results),
		Results:        results,
	}

	// Save result (overwrite = retake)
	_, err = config.Firestore.Collection("posttestResults").Doc(docID).Set(ctx, map[string]interface{}{
		"uid":            uid,
		"algorithm":      algorithm,
		"score":          score,
		"totalQuestions": len(results),
		"answers":        submission.Answers,
		"results":        results,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save posttest result: %v", err)
	}

	// Delete progress document
	_, _ = config.Firestore.Collection("posttestProgress").Doc(docID).Delete(ctx)

	if err := RefreshUserProfileProgress(uid); err != nil {
		fmt.Printf("Warning: failed to refresh user profile progress after posttest submit: %v\n", err)
	}

	return gradingResult, nil
}

// gradeOnePosttestQuestion grades a single question and returns the result with correct answer
func gradeOnePosttestQuestion(q models.QuizQuestion, answer models.PosttestAnswerDTO, hasAnswer bool) models.PosttestQuestionResult {
	result := models.PosttestQuestionResult{
		QuestionId: q.ID,
		Type:       q.Type,
	}

	if q.Question == nil {
		return result
	}

	mapData, ok := q.Question.(map[string]interface{})
	if !ok {
		return result
	}

	switch q.Type {
	case "multiple_choice":
		var mc models.MultipleChoiceQuestion
		_ = mapstructure.Decode(mapData, &mc)

		correctId := string(rune('a' + mc.CorrectChoiceIndex))
		result.CorrectChoiceId = correctId

		if hasAnswer && answer.SelectedChoiceId == correctId {
			result.IsCorrect = true
		}

	case "fill_blank":
		var fb models.FillQuestion
		_ = mapstructure.Decode(mapData, &fb)

		result.CorrectAnswer = fb.CorrectAnswer

		if hasAnswer {
			userAns := strings.TrimSpace(strings.ToLower(answer.FilledAnswer))
			correctAns := strings.TrimSpace(strings.ToLower(fb.CorrectAnswer))
			result.IsCorrect = userAns == correctAns
		}

	case "ordering":
		var ord models.OrderingQuestion
		_ = mapstructure.Decode(mapData, &ord)

		_, labelToIDs := mapOrderingItemsToIDs(ord.Items, ord.CanvasData)

		// Build correct order by label→position mapping
		correctOrder := make([]string, len(ord.CorrectOrder))
		labelUseCount := make(map[string]int)
		for _, co := range ord.CorrectOrder {
			ids := labelToIDs[co.Label]
			useIdx := labelUseCount[co.Label]
			if co.Position >= 0 && co.Position < len(correctOrder) && useIdx < len(ids) {
				correctOrder[co.Position] = ids[useIdx]
				labelUseCount[co.Label] = useIdx + 1
			}
		}
		result.CorrectOrder = correctOrder

		if hasAnswer && len(answer.OrderedItems) == len(correctOrder) {
			allMatch := true
			for i, id := range answer.OrderedItems {
				if id != correctOrder[i] {
					allMatch = false
					break
				}
			}
			result.IsCorrect = allMatch
		}
	}

	return result
}

// ── Status ──────────────────────────────────────────────────────

func GetPosttestStatus(uid string, algorithm string) (*models.PosttestStatus, error) {
	ctx := context.Background()
	docID := postDocID(uid, algorithm)

	// Check completed
	resultDoc, err := config.Firestore.Collection("posttestResults").Doc(docID).Get(ctx)
	if err == nil {
		data := resultDoc.Data()
		score, _ := data["score"].(int64)
		total, _ := data["totalQuestions"].(int64)

		return &models.PosttestStatus{
			Completed: true,
			Score:     int(score),
			Total:     int(total),
		}, nil
	}

	// Check in-progress
	progressDoc, err := config.Firestore.Collection("posttestProgress").Doc(docID).Get(ctx)
	if err == nil {
		var progress models.PosttestProgress
		if err := progressDoc.DataTo(&progress); err == nil {
			return &models.PosttestStatus{
				InProgress:    true,
				AnsweredCount: progress.AnsweredCount,
				Total:         len(progress.QuestionIds),
			}, nil
		}
	}

	return &models.PosttestStatus{}, nil
}

// ── Internal helpers ────────────────────────────────────────────

type parsedPosttestQuestion struct {
	quiz models.QuizQuestion
}

func mapOrderingItemsToIDs(items []models.OrderingItem, canvasData *models.CanvasData) ([]models.PosttestOrderItemDTO, map[string][]string) {
	result := make([]models.PosttestOrderItemDTO, len(items))
	labelToIDs := make(map[string][]string)

	if canvasData == nil {
		for i, item := range items {
			id := fmt.Sprintf("i%d", i)
			result[i] = models.PosttestOrderItemDTO{ID: id, Label: item.Label}
			labelToIDs[item.Label] = append(labelToIDs[item.Label], id)
		}
		return result, labelToIDs
	}

	usedNodeIdx := make(map[int]bool)
	for i, item := range items {
		id := fmt.Sprintf("i%d", i)

		for nodeIdx, node := range canvasData.Nodes {
			if usedNodeIdx[nodeIdx] {
				continue
			}
			data, ok := node["data"].(map[string]interface{})
			if !ok {
				continue
			}
			label, ok := data["label"].(string)
			if !ok || label != item.Label {
				continue
			}
			nodeID, ok := node["id"].(string)
			if !ok || nodeID == "" {
				continue
			}

			id = nodeID
			usedNodeIdx[nodeIdx] = true
			break
		}

		result[i] = models.PosttestOrderItemDTO{ID: id, Label: item.Label}
		labelToIDs[item.Label] = append(labelToIDs[item.Label], id)
	}

	return result, labelToIDs
}

func selectPosttestQuestions(all []parsedPosttestQuestion, count int) []parsedPosttestQuestion {
	types := []string{"multiple_choice", "fill_blank", "ordering"}

	byType := make(map[string][]parsedPosttestQuestion)
	for _, pq := range all {
		byType[pq.quiz.Type] = append(byType[pq.quiz.Type], pq)
	}

	selected := make([]parsedPosttestQuestion, 0, count)
	usedIDs := make(map[string]bool)

	for _, t := range types {
		pool := byType[t]
		if len(pool) > 0 {
			pick := pool[rand.Intn(len(pool))]
			selected = append(selected, pick)
			usedIDs[pick.quiz.ID] = true
		}
	}

	remaining := make([]parsedPosttestQuestion, 0)
	for _, pq := range all {
		if !usedIDs[pq.quiz.ID] {
			remaining = append(remaining, pq)
		}
	}

	rand.Shuffle(len(remaining), func(i, j int) {
		remaining[i], remaining[j] = remaining[j], remaining[i]
	})

	for _, pq := range remaining {
		if len(selected) >= count {
			break
		}
		selected = append(selected, pq)
	}

	rand.Shuffle(len(selected), func(i, j int) {
		selected[i], selected[j] = selected[j], selected[i]
	})

	return selected
}

func fetchPosttestQuestionsByIDs(ctx context.Context, ids []string) ([]models.PosttestQuestionDTO, error) {
	questions := make([]models.PosttestQuestionDTO, 0, len(ids))

	for _, id := range ids {
		doc, err := config.Firestore.Collection("quizQuestions").Doc(id).Get(ctx)
		if err != nil {
			continue
		}

		var q models.QuizQuestion
		if err := doc.DataTo(&q); err != nil {
			continue
		}
		q.ID = doc.Ref.ID
		dto := transformPosttestToDTO(q)
		if dto != nil {
			questions = append(questions, *dto)
		}
	}

	return questions, nil
}

// transformPosttestToDTO — strips correct answers
func transformPosttestToDTO(q models.QuizQuestion) *models.PosttestQuestionDTO {
	dto := &models.PosttestQuestionDTO{
		ID:            q.ID,
		Type:          q.Type,
		Title:         q.Title,
		Text:          q.Title,
		QuestionImage: q.QuestionImage,
	}

	if q.Question == nil {
		return dto
	}

	mapData, ok := q.Question.(map[string]interface{})
	if !ok {
		return dto
	}

	switch q.Type {
	case "multiple_choice":
		var mc models.MultipleChoiceQuestion
		_ = mapstructure.Decode(mapData, &mc)

		choices := make([]models.PosttestChoiceDTO, len(mc.Choices))
		for i, c := range mc.Choices {
			choices[i] = models.PosttestChoiceDTO{
				ID:    string(rune('a' + i)),
				Label: string(rune('A' + i)),
				Text:  c.Label,
			}
		}

		mcDTO := models.PosttestMultipleChoiceDTO{}
		mcDTO.MultipleChoice.Choices = choices
		dto.Question = mcDTO

	case "fill_blank":
		dto.Question = models.PosttestFillBlankDTO{}

	case "ordering":
		var ord models.OrderingQuestion
		_ = mapstructure.Decode(mapData, &ord)

		items, _ := mapOrderingItemsToIDs(ord.Items, ord.CanvasData)

		ordDTO := models.PosttestOrderingDTO{
			Items: items,
		}

		// Pass through canvasData if present (tree/graph ordering)
		if ord.CanvasData != nil {
			ordDTO.CanvasData = ord.CanvasData
		}

		dto.Question = ordDTO
	}

	return dto
}

func posttestTitleCase(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
