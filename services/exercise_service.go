package services

import (
	"algoplayground/config"
	"algoplayground/models"
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// CreateExercises batch inserts multiple exercises
func CreateExercises(exercises []models.Exercise) error {
	ctx := context.Background()
	batch := config.Firestore.Batch()

	for _, ex := range exercises {
		var docRef *firestore.DocumentRef
		if ex.ID != "" {
			docRef = config.Firestore.Collection("exercises").Doc(ex.ID)
		} else {
			// Generate a new ID if not provided
			docRef = config.Firestore.Collection("exercises").NewDoc()
			ex.ID = docRef.ID
		}
		batch.Set(docRef, ex)
	}

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit batch created exercises: %v", err)
	}

	return nil
}

// GetExercises retrieves all exercises from Firestore
func GetExercises() ([]models.Exercise, error) {
	ctx := context.Background()
	var exercises []models.Exercise

	iter := config.Firestore.Collection("exercises").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate over exercises: %v", err)
		}

		var ex models.Exercise
		if err := doc.DataTo(&ex); err != nil {
			return nil, fmt.Errorf("failed to map exercise data: %v", err)
		}
		exercises = append(exercises, ex)
	}

	return exercises, nil
}
