package services

import (
	"algoplayground/config"
	"algoplayground/models"
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"
)

func GetQuizzes(algorithm string, typeQuiz string) ([]models.QuizQuestion, error) {
	ctx := context.Background()
	var quizzes []models.QuizQuestion

	// Query quizQuestions collection
	query := config.Firestore.Collection("quizQuestions").
		Where("algorithm", "==", algorithm).
		Where("typeQuiz", "==", typeQuiz)

	iter := query.Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quizzes: %v", err)
	}

	for _, doc := range docs {
		var q models.QuizQuestion
		if err := doc.DataTo(&q); err != nil {
			fmt.Printf("Error mapping quiz question document %s: %v\n", doc.Ref.ID, err)
			continue
		}
		// Ensure ID matches document ID
		q.ID = doc.Ref.ID

		// Parse the polymorphic Question field based on Type
		if q.Question != nil {
			mapData, ok := q.Question.(map[string]interface{})
			if ok {
				switch q.Type {
				case "multiple_choice":
					var mc models.MultipleChoiceQuestion
					_ = mapstructure.Decode(mapData, &mc)
					q.Question = mc
				case "fill_blank":
					var fill models.FillQuestion
					_ = mapstructure.Decode(mapData, &fill)
					q.Question = fill
				case "ordering":
					var ord models.OrderingQuestion
					_ = mapstructure.Decode(mapData, &ord)
					q.Question = ord
				}
			}
		}

		quizzes = append(quizzes, q)
	}

	return quizzes, nil
}
