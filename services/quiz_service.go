package services

import (
	"algoplayground/config"
	"algoplayground/models"
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
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

// CreateQuizzes batch inserts multiple quiz questions
func CreateQuizzes(quizzes []models.QuizQuestion) error {
	ctx := context.Background()
	batch := config.Firestore.Batch()

	for _, quiz := range quizzes {
		var docRef *firestore.DocumentRef
		if quiz.ID != "" {
			docRef = config.Firestore.Collection("quizQuestions").Doc(quiz.ID)
		} else {
			// Generate a new ID if not provided
			docRef = config.Firestore.Collection("quizQuestions").NewDoc()
			quiz.ID = docRef.ID
		}
		batch.Set(docRef, quiz)
	}

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit batch created quizzes: %v", err)
	}

	return nil
}
