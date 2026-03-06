package models

import "time"

// User represents the main user document in Firestore
type User struct {
	ID        string    `json:"id" firestore:"id"` // Firebase UID
	ImageURL  string    `json:"imageUrl" firestore:"imageUrl"`
	UpdatedAt time.Time `json:"updatedAt" firestore:"updatedAt"`
}

// UserProgress tracks overall learning progress
type UserProgress struct {
	UserID        string  `json:"userId" firestore:"userId"`
	TotalProgress float64 `json:"totalProgress" firestore:"totalProgress"`
	PretestScore  float64 `json:"pretestScore" firestore:"pretestScore"`
	PosttestScore float64 `json:"posttestScore" firestore:"posttestScore"`
}

// UserAlgorithmSection groups algorithms by category (e.g. "Linear DS", "Tree")
type UserAlgorithmSection struct {
	UserID string          `json:"userid" firestore:"userid"`
	Title  string          `json:"title" firestore:"title"`
	Item   []UserAlgorithm `json:"item" firestore:"item"`
}

// UserAlgorithm represents a single algorithm entry
type UserAlgorithm struct {
	Slug     string             `json:"slug" firestore:"slug"`
	Title    string             `json:"title" firestore:"title"`
	Progress []UserTestProgress `json:"progress" firestore:"progress"`
}

// UserTestProgress tracks pretest/posttest status for a specific algorithm
type UserTestProgress struct {
	Pretest  []UserProgressPretest  `json:"pretest" firestore:"pretest"`
	Posttest []UserProgressPosttest `json:"posttest" firestore:"posttest"`
}

type UserProgressPretest struct {
	PretestScore  int    `json:"pretestScore" firestore:"pretestScore"`
	PretestStatus string `json:"pretestStatus" firestore:"pretestStatus"` // active/ locked/ Completed
}

type UserProgressPosttest struct {
	PosttestScore  int    `json:"posttestScore" firestore:"posttestScore"`
	PosttestStatus string `json:"posttestStatus" firestore:"posttestStatus"` // active/ locked/ Completed
}

// UserCategoryAlgoProgressInProfile tracks progress per algorithm category
type UserCategoryAlgoProgressInProfile struct {
	UserID    string `json:"userId" firestore:"userId"`
	Linear    int    `json:"linear" firestore:"linear"`
	Trees     int    `json:"trees" firestore:"trees"`
	Graph     int    `json:"graph" firestore:"graph"`
	Sorting   int    `json:"sorting" firestore:"sorting"`
	Searching int    `json:"searching" firestore:"searching"`
}

// UserTestTotalProgressInProfile tracks total progress per test type
type UserTestTotalProgressInProfile struct {
	UserID   string  `json:"userId" firestore:"userId"`
	Total    float64 `json:"total" firestore:"total"`
	Pretest  float64 `json:"pretest" firestore:"pretest"`
	Posttest float64 `json:"posttest" firestore:"posttest"`
}
