package services

import (
	"algoplayground/config"
	"context"
	"fmt"
)

// AlgorithmProgress holds pretest + posttest status for one algorithm
type AlgorithmProgress struct {
	Pretest  TestStatus `json:"pretest"`
	Posttest TestStatus `json:"posttest"`
}

// TestStatus is the status of a single test (pretest or posttest)
type TestStatus struct {
	Status        string `json:"status"` // locked | active | completed
	Score         int    `json:"score,omitempty"`
	TotalCount    int    `json:"totalCount"`
	AnsweredCount int    `json:"answeredCount,omitempty"`
}

// GetAllProgress returns pretest + posttest status for every algorithm the user has interacted with
func GetAllProgress(uid string) (map[string]AlgorithmProgress, error) {
	ctx := context.Background()
	result := make(map[string]AlgorithmProgress)

	// Helper to ensure an entry exists
	getOrCreate := func(algo string) AlgorithmProgress {
		if p, ok := result[algo]; ok {
			return p
		}
		return AlgorithmProgress{
			Pretest:  TestStatus{Status: "locked", TotalCount: 5},
			Posttest: TestStatus{Status: "locked", TotalCount: 5},
		}
	}

	// 1) pretestResults → completed pretests
	pretestResults, err := config.Firestore.Collection("pretestResults").
		Where("uid", "==", uid).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pretest results: %v", err)
	}
	for _, doc := range pretestResults {
		data := doc.Data()
		algo, _ := data["algorithm"].(string)
		if algo == "" {
			continue
		}
		p := getOrCreate(algo)
		score, _ := data["score"].(int64)
		total, _ := data["totalQuestions"].(int64)
		if total == 0 {
			total = 5
		}
		p.Pretest = TestStatus{
			Status:     "completed",
			Score:      int(score),
			TotalCount: int(total),
		}
		result[algo] = p
	}

	// 2) pretestProgress → in-progress pretests (only if not already completed)
	pretestProgress, err := config.Firestore.Collection("pretestProgress").
		Where("uid", "==", uid).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pretest progress: %v", err)
	}
	for _, doc := range pretestProgress {
		data := doc.Data()
		algo, _ := data["algorithm"].(string)
		if algo == "" {
			continue
		}
		p := getOrCreate(algo)
		if p.Pretest.Status == "completed" {
			result[algo] = p
			continue // already completed, skip
		}
		answeredCount, _ := data["answeredCount"].(int64)
		questionIds, _ := data["questionIds"].([]interface{})
		totalCount := len(questionIds)
		if totalCount == 0 {
			totalCount = 5
		}
		p.Pretest = TestStatus{
			Status:        "active",
			AnsweredCount: int(answeredCount),
			TotalCount:    totalCount,
		}
		result[algo] = p
	}

	// 3) posttestResults → completed posttests
	posttestResults, err := config.Firestore.Collection("posttestResults").
		Where("uid", "==", uid).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posttest results: %v", err)
	}
	for _, doc := range posttestResults {
		data := doc.Data()
		algo, _ := data["algorithm"].(string)
		if algo == "" {
			continue
		}
		p := getOrCreate(algo)
		score, _ := data["score"].(int64)
		total, _ := data["totalQuestions"].(int64)
		if total == 0 {
			total = 5
		}
		p.Posttest = TestStatus{
			Status:     "completed",
			Score:      int(score),
			TotalCount: int(total),
		}
		result[algo] = p
	}

	// 4) posttestProgress → in-progress posttests (only if not already completed)
	posttestProgress, err := config.Firestore.Collection("posttestProgress").
		Where("uid", "==", uid).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posttest progress: %v", err)
	}
	for _, doc := range posttestProgress {
		data := doc.Data()
		algo, _ := data["algorithm"].(string)
		if algo == "" {
			continue
		}
		p := getOrCreate(algo)
		if p.Posttest.Status == "completed" {
			result[algo] = p
			continue
		}
		answeredCount, _ := data["answeredCount"].(int64)
		questionIds, _ := data["questionIds"].([]interface{})
		totalCount := len(questionIds)
		if totalCount == 0 {
			totalCount = 5
		}
		p.Posttest = TestStatus{
			Status:        "active",
			AnsweredCount: int(answeredCount),
			TotalCount:    totalCount,
		}
		result[algo] = p
	}

	return result, nil
}
