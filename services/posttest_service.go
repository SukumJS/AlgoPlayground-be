package services

import (
	"algoplayground/config"
	"algoplayground/models"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	posttestReminderCollection = "posttestReminderState"
)

var (
	ErrInvalidAlgorithm      = errors.New("invalid algorithm")
	ErrInvalidReminderSource = errors.New("invalid reminder source")
	ErrReminderResetDisabled = errors.New("reminder reset endpoint is disabled")
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
			fromPretestSet := make(map[string]bool, len(progress.FromPretestIds))
			for _, id := range progress.FromPretestIds {
				fromPretestSet[id] = true
			}
			questions, err := fetchPosttestQuestionsByIDs(ctx, progress.QuestionIds, fromPretestSet)
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

	// Replace some posttest MC questions with wrong pretest questions (total stays 5)
	pretestWrong := fetchPretestWrongQuestionsForPosttest(ctx, uid, algorithm)
	if len(pretestWrong) > 0 {
		mcIndices := []int{}
		for i, pq := range selected {
			if pq.quiz.Type == "multiple_choice" {
				mcIndices = append(mcIndices, i)
			}
		}
		replaceCount := len(pretestWrong)
		if replaceCount > len(mcIndices) {
			replaceCount = len(mcIndices)
		}
		removeSet := make(map[int]bool, replaceCount)
		for _, idx := range mcIndices[:replaceCount] {
			removeSet[idx] = true
		}
		kept := make([]parsedPosttestQuestion, 0, 5)
		for i, pq := range selected {
			if !removeSet[i] {
				kept = append(kept, pq)
			}
		}
		kept = append(kept, pretestWrong[:replaceCount]...)
		rand.Shuffle(len(kept), func(i, j int) { kept[i], kept[j] = kept[j], kept[i] })
		selected = kept
	}

	dtos := make([]models.PosttestQuestionDTO, 0, len(selected))
	questionIds := make([]string, 0, len(selected))
	fromPretestIds := make([]string, 0)
	for _, pq := range selected {
		dto := transformPosttestToDTO(pq.quiz)
		if dto != nil {
			dto.FromPretest = pq.fromPretest
			dtos = append(dtos, *dto)
			questionIds = append(questionIds, pq.quiz.ID)
			if pq.fromPretest {
				fromPretestIds = append(fromPretestIds, pq.quiz.ID)
			}
		}
	}

	// 3) Create progress document
	_, err = config.Firestore.Collection("posttestProgress").Doc(docID).Set(ctx, models.PosttestProgress{
		UID:            uid,
		Algorithm:      algorithm,
		QuestionIds:    questionIds,
		Answers:        []models.PosttestAnswerDTO{},
		AnsweredCount:  0,
		FromPretestIds: fromPretestIds,
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

	if _, err := markPosttestReminderSeenInternal(ctx, uid, algorithm, "posttest-completed"); err != nil {
		fmt.Printf("Warning: failed to mark posttest reminder as seen after submit: %v\n", err)
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
	if !isSupportedAlgorithmSlug(algorithm) {
		return nil, ErrInvalidAlgorithm
	}

	ctx := context.Background()
	docID := postDocID(uid, algorithm)
	now := time.Now().UTC()

	var resultSnap *posttestResultSnapshot
	resultDoc, err := config.Firestore.Collection("posttestResults").Doc(docID).Get(ctx)
	if err == nil {
		data := resultDoc.Data()
		resultSnap = &posttestResultSnapshot{
			Score:     toInt(data["score"]),
			Total:     toInt(data["totalQuestions"]),
			UpdatedAt: resultDoc.UpdateTime.UTC(),
		}
	} else if !isNotFoundError(err) {
		return nil, fmt.Errorf("failed to fetch posttest result: %v", err)
	}

	var progressSnap *posttestProgressSnapshot
	progressDoc, err := config.Firestore.Collection("posttestProgress").Doc(docID).Get(ctx)
	if err == nil {
		var progress models.PosttestProgress
		if err := progressDoc.DataTo(&progress); err == nil {
			progressSnap = &posttestProgressSnapshot{
				AnsweredCount: progress.AnsweredCount,
				Total:         len(progress.QuestionIds),
				UpdatedAt:     progressDoc.UpdateTime.UTC(),
			}
		}
	} else if !isNotFoundError(err) {
		return nil, fmt.Errorf("failed to fetch posttest progress: %v", err)
	}

	reminder, err := getPosttestReminderRecord(ctx, uid, algorithm)
	if err != nil {
		return nil, err
	}

	status := composePosttestStatus(algorithm, resultSnap, progressSnap, reminder, now)
	return &status, nil
}

// MarkPosttestReminderSeen records that the reminder modal was dismissed.
func MarkPosttestReminderSeen(uid string, algorithm string, source string) (*models.PosttestReminderState, error) {
	if !isSupportedAlgorithmSlug(algorithm) {
		return nil, ErrInvalidAlgorithm
	}

	normalizedSource := strings.TrimSpace(strings.ToLower(source))
	if normalizedSource == "" {
		normalizedSource = "maybe-later"
	}
	if !isValidReminderSource(normalizedSource) {
		return nil, ErrInvalidReminderSource
	}

	return markPosttestReminderSeenInternal(context.Background(), uid, algorithm, normalizedSource)
}

// ResetPosttestReminder resets reminder state for QA/dev only.
func ResetPosttestReminder(uid string, algorithm string) (*models.PosttestReminderState, error) {
	if !isReminderResetEnabled() {
		return nil, ErrReminderResetDisabled
	}

	if !isSupportedAlgorithmSlug(algorithm) {
		return nil, ErrInvalidAlgorithm
	}

	ctx := context.Background()
	now := time.Now().UTC()
	docID := postDocID(uid, algorithm)

	_, err := config.Firestore.Collection(posttestReminderCollection).Doc(docID).Set(ctx, map[string]interface{}{
		"uid":             uid,
		"algorithm":       algorithm,
		"reminderShown":   false,
		"reminderShownAt": nil,
		"source":          "reminder-reset",
		"updatedAt":       now,
	}, firestore.MergeAll)
	if err != nil {
		return nil, fmt.Errorf("failed to reset posttest reminder: %v", err)
	}

	return &models.PosttestReminderState{
		Algorithm:       algorithm,
		ReminderShown:   false,
		ReminderShownAt: nil,
		UpdatedAt:       now,
	}, nil
}

func markPosttestReminderSeenInternal(ctx context.Context, uid string, algorithm string, source string) (*models.PosttestReminderState, error) {
	now := time.Now().UTC()
	docID := postDocID(uid, algorithm)
	docRef := config.Firestore.Collection(posttestReminderCollection).Doc(docID)

	var shownAt *time.Time
	doc, err := docRef.Get(ctx)
	if err == nil {
		if existing, err := reminderRecordFromDoc(doc); err == nil && existing.ReminderShownAt != nil {
			t := existing.ReminderShownAt.UTC()
			shownAt = &t
		}
	} else if !isNotFoundError(err) {
		return nil, fmt.Errorf("failed to fetch posttest reminder: %v", err)
	}

	if shownAt == nil {
		t := now
		shownAt = &t
	}

	_, err = docRef.Set(ctx, map[string]interface{}{
		"uid":             uid,
		"algorithm":       algorithm,
		"reminderShown":   true,
		"reminderShownAt": shownAt,
		"source":          source,
		"updatedAt":       now,
	}, firestore.MergeAll)
	if err != nil {
		return nil, fmt.Errorf("failed to mark posttest reminder as seen: %v", err)
	}

	return &models.PosttestReminderState{
		Algorithm:       algorithm,
		ReminderShown:   true,
		ReminderShownAt: shownAt,
		UpdatedAt:       now,
	}, nil
}

type posttestResultSnapshot struct {
	Score     int
	Total     int
	UpdatedAt time.Time
}

type posttestProgressSnapshot struct {
	AnsweredCount int
	Total         int
	UpdatedAt     time.Time
}

func composePosttestStatus(algorithm string, resultSnap *posttestResultSnapshot, progressSnap *posttestProgressSnapshot, reminder *models.PosttestReminderRecord, now time.Time) models.PosttestStatus {
	zero := 0
	out := models.PosttestStatus{
		Algorithm:       algorithm,
		Completed:       false,
		InProgress:      false,
		Score:           nil,
		Total:           nil,
		AnsweredCount:   &zero,
		ReminderShown:   false,
		ReminderShownAt: nil,
		UpdatedAt:       time.Time{},
	}

	if progressSnap != nil {
		out.InProgress = true
		out.AnsweredCount = intPtr(progressSnap.AnsweredCount)
		out.Total = intPtr(progressSnap.Total)
		out.UpdatedAt = maxTimeOrDefault(out.UpdatedAt, progressSnap.UpdatedAt)
	}

	if resultSnap != nil {
		out.Completed = true
		out.InProgress = false
		out.Score = intPtr(resultSnap.Score)
		out.Total = intPtr(resultSnap.Total)
		out.AnsweredCount = nil
		out.UpdatedAt = maxTimeOrDefault(out.UpdatedAt, resultSnap.UpdatedAt)
	}

	if reminder != nil {
		out.ReminderShown = reminder.ReminderShown
		out.ReminderShownAt = reminder.ReminderShownAt
		out.UpdatedAt = maxTimeOrDefault(out.UpdatedAt, reminder.UpdatedAt)
	}

	if out.Completed {
		out.ReminderShown = true
		if out.ReminderShownAt == nil && resultSnap != nil {
			t := resultSnap.UpdatedAt
			out.ReminderShownAt = &t
		}
	}

	if out.UpdatedAt.IsZero() {
		out.UpdatedAt = now
	}

	return out
}

func getPosttestReminderRecord(ctx context.Context, uid string, algorithm string) (*models.PosttestReminderRecord, error) {
	docID := postDocID(uid, algorithm)
	doc, err := config.Firestore.Collection(posttestReminderCollection).Doc(docID).Get(ctx)
	if isNotFoundError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posttest reminder: %v", err)
	}

	record, err := reminderRecordFromDoc(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse posttest reminder: %v", err)
	}

	return record, nil
}

func reminderRecordFromDoc(doc *firestore.DocumentSnapshot) (*models.PosttestReminderRecord, error) {
	var record models.PosttestReminderRecord
	if err := doc.DataTo(&record); err != nil {
		return nil, err
	}

	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = doc.UpdateTime.UTC()
	} else {
		record.UpdatedAt = record.UpdatedAt.UTC()
	}

	if record.ReminderShownAt != nil {
		t := record.ReminderShownAt.UTC()
		record.ReminderShownAt = &t
	}

	return &record, nil
}

func isSupportedAlgorithmSlug(algorithm string) bool {
	_, ok := algorithmCategoryCatalog[algorithm]
	return ok
}

func isValidReminderSource(source string) bool {
	switch source {
	case "maybe-later", "posttest-completed", "system-sync":
		return true
	default:
		return false
	}
}

func isReminderResetEnabled() bool {
	// Explicit opt-in only to avoid accidental enablement in production.
	return strings.EqualFold(strings.TrimSpace(os.Getenv("POSTTEST_REMINDER_RESET_ENABLED")), "true")
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func maxTimeOrDefault(a, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	return maxTime(a, b)
}

func intPtr(v int) *int {
	value := v
	return &value
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float32:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func isNotFoundError(err error) bool {
	return status.Code(err) == codes.NotFound
}

// ── Internal helpers ────────────────────────────────────────────

type parsedPosttestQuestion struct {
	quiz        models.QuizQuestion
	fromPretest bool
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

// fetchPretestWrongQuestionsForPosttest returns up to 2 questions the user answered
// incorrectly in their pretest. Returns nil if no pretest result exists.
func fetchPretestWrongQuestionsForPosttest(ctx context.Context, uid, algorithm string) []parsedPosttestQuestion {
	docID := progressDocID(uid, algorithm)
	doc, err := config.Firestore.Collection("pretestResults").Doc(docID).Get(ctx)
	if err != nil {
		return nil
	}

	data := doc.Data()
	rawResults, ok := data["results"].([]interface{})
	if !ok {
		return nil
	}

	wrongIds := []string{}
	for _, r := range rawResults {
		m, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		isCorrect, _ := m["isCorrect"].(bool)
		qid, _ := m["questionId"].(string)
		if !isCorrect && qid != "" {
			wrongIds = append(wrongIds, qid)
		}
	}

	if len(wrongIds) == 0 {
		return nil
	}

	rand.Shuffle(len(wrongIds), func(i, j int) { wrongIds[i], wrongIds[j] = wrongIds[j], wrongIds[i] })
	count := 1
	if len(wrongIds) >= 2 {
		count = 2
	}
	wrongIds = wrongIds[:count]

	result := make([]parsedPosttestQuestion, 0, count)
	for _, qid := range wrongIds {
		qDoc, err := config.Firestore.Collection("quizQuestions").Doc(qid).Get(ctx)
		if err != nil {
			continue
		}
		var q models.QuizQuestion
		if err := qDoc.DataTo(&q); err != nil {
			continue
		}
		q.ID = qDoc.Ref.ID
		result = append(result, parsedPosttestQuestion{quiz: q, fromPretest: true})
	}
	return result
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

func fetchPosttestQuestionsByIDs(ctx context.Context, ids []string, fromPretestSet map[string]bool) ([]models.PosttestQuestionDTO, error) {
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
			dto.FromPretest = fromPretestSet[id]
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
