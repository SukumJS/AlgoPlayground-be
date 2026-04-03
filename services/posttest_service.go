package services

import (
	"algoplayground/config"
	"algoplayground/models"
	"context"
	"fmt"
	"math/rand"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// GetPosttestByAlgorithm fetches posttest questions from Firestore,
// selects 5 random questions (≥1 of each type), and returns DTOs.
func GetPosttestByAlgorithm(algorithm string) (*models.PosttestResponse, error) {
	ctx := context.Background()

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

	// Parse all questions
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

	// Random selection: 5 questions, ≥1 of each type
	selected := selectPosttestQuestions(allQuestions, 5)

	// Transform to DTOs
	dtos := make([]models.PosttestQuestionDTO, 0, len(selected))
	for _, pq := range selected {
		dto := transformPosttestToDTO(pq.quiz)
		if dto != nil {
			dtos = append(dtos, *dto)
		}
	}

	title := "Post Test Of " + posttestTitleCase(algorithm)

	return &models.PosttestResponse{
		ID:        "posttest-" + algorithm,
		Title:     title,
		Questions: dtos,
	}, nil
}

// ── Internal types ──────────────────────────────────────────────

type parsedPosttestQuestion struct {
	quiz models.QuizQuestion
}

// ── Random selection (same logic as frontend mock) ──────────────

func selectPosttestQuestions(all []parsedPosttestQuestion, count int) []parsedPosttestQuestion {
	types := []string{"multiple_choice", "fill_blank", "ordering"}

	// Group by type
	byType := make(map[string][]parsedPosttestQuestion)
	for _, pq := range all {
		byType[pq.quiz.Type] = append(byType[pq.quiz.Type], pq)
	}

	selected := make([]parsedPosttestQuestion, 0, count)
	usedIDs := make(map[string]bool)

	// Pick at least 1 of each type
	for _, t := range types {
		pool := byType[t]
		if len(pool) > 0 {
			pick := pool[rand.Intn(len(pool))]
			selected = append(selected, pick)
			usedIDs[pick.quiz.ID] = true
		}
	}

	// Fill remaining slots from unused questions
	remaining := make([]parsedPosttestQuestion, 0)
	for _, pq := range all {
		if !usedIDs[pq.quiz.ID] {
			remaining = append(remaining, pq)
		}
	}

	// Shuffle remaining
	rand.Shuffle(len(remaining), func(i, j int) {
		remaining[i], remaining[j] = remaining[j], remaining[i]
	})

	for _, pq := range remaining {
		if len(selected) >= count {
			break
		}
		selected = append(selected, pq)
	}

	// Shuffle final selection
	rand.Shuffle(len(selected), func(i, j int) {
		selected[i], selected[j] = selected[j], selected[i]
	})

	return selected
}

// ── DTO transformation ──────────────────────────────────────────

func transformPosttestToDTO(q models.QuizQuestion) *models.PosttestQuestionDTO {
	dto := &models.PosttestQuestionDTO{
		ID:            q.ID,
		Type:          q.Type,
		Title:         q.Title,
		Text:          q.Title, // title is the question text
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

		correctId := string(rune('a' + mc.CorrectChoiceIndex))

		mcDTO := models.PosttestMultipleChoiceDTO{}
		mcDTO.MultipleChoice.Choices = choices
		mcDTO.MultipleChoice.CorrectChoiceId = correctId
		dto.Question = mcDTO

	case "fill_blank":
		var fb models.FillQuestion
		_ = mapstructure.Decode(mapData, &fb)

		dto.Question = models.PosttestFillBlankDTO{
			CorrectAnswer: fb.CorrectAnswer,
		}

	case "ordering":
		var ord models.OrderingQuestion
		_ = mapstructure.Decode(mapData, &ord)

		items := make([]models.PosttestOrderItemDTO, len(ord.Items))
		for i, item := range ord.Items {
			items[i] = models.PosttestOrderItemDTO{
				ID:    fmt.Sprintf("i%d", i),
				Label: item.Label,
			}
		}

		// Build correct order from CorrectOrder positions
		correctOrder := make([]string, len(ord.CorrectOrder))
		for _, co := range ord.CorrectOrder {
			if co.Position >= 0 && co.Position < len(items) {
				correctOrder[co.Position] = items[co.Position].ID
			}
		}

		// Match correctOrder by label (Firestore correctOrder has label+position)
		// Build label→id map
		labelToID := make(map[string]string)
		for _, item := range items {
			labelToID[item.Label] = item.ID
		}

		correctOrderByLabel := make([]string, len(ord.CorrectOrder))
		for _, co := range ord.CorrectOrder {
			if id, ok := labelToID[co.Label]; ok {
				correctOrderByLabel[co.Position] = id
			}
		}

		dto.Question = models.PosttestOrderingDTO{
			Items:        items,
			CorrectOrder: correctOrderByLabel,
		}
	}

	return dto
}

// posttestTitleCase converts kebab-case to Title Case
func posttestTitleCase(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
