package services

import (
	"algoplayground/config"
	"algoplayground/models"
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

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
