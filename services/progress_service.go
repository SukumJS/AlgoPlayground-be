package services

import (
	"algoplayground/config"
	"algoplayground/models"
	"context"
	"fmt"
	"math"

	"cloud.google.com/go/firestore"
)

var algorithmCategoryCatalog = map[string]string{
	"array":                 "linear",
	"doubly-linked-list":    "linear",
	"singly-linked-list":    "linear",
	"stack":                 "linear",
	"queue":                 "linear",
	"binary-tree-inorder":   "trees",
	"binary-tree-preorder":  "trees",
	"binary-tree-postorder": "trees",
	"binary-search-tree":    "trees",
	"avl-tree":              "trees",
	"min-heap":              "trees",
	"max-heap":              "trees",
	"breadth-first-search":  "graph",
	"depth-first-search":    "graph",
	"dijkstra":              "graph",
	"bellman-ford":          "graph",
	"prims":                 "graph",
	"kruskals":              "graph",
	"bubble-sort":           "sorting",
	"selection-sort":        "sorting",
	"insertion-sort":        "sorting",
	"merge-sort":            "sorting",
	"queue-sort":            "sorting",
	"linear-search":         "searching",
	"binary-search":         "searching",
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	default:
		return 0
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

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

// RefreshUserProfileProgress recalculates users.progress and users.categoryAlgoProgress from result collections.
func RefreshUserProfileProgress(uid string) error {
	ctx := context.Background()

	preDocs, err := config.Firestore.Collection("pretestResults").Where("uid", "==", uid).Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("failed to fetch pretest results: %v", err)
	}

	postDocs, err := config.Firestore.Collection("posttestResults").Where("uid", "==", uid).Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("failed to fetch posttest results: %v", err)
	}

	preByAlgo := make(map[string]bool)
	postByAlgo := make(map[string]bool)

	preScoreSum := 0.0
	preScoreCount := 0.0
	for _, doc := range preDocs {
		data := doc.Data()
		algo, _ := data["algorithm"].(string)
		if algo != "" {
			preByAlgo[algo] = true
		}

		score := toFloat(data["score"])
		total := toFloat(data["totalQuestions"])
		if total > 0 {
			preScoreSum += (score / total) * 100
			preScoreCount++
		}
	}

	postScoreSum := 0.0
	postScoreCount := 0.0
	for _, doc := range postDocs {
		data := doc.Data()
		algo, _ := data["algorithm"].(string)
		if algo != "" {
			postByAlgo[algo] = true
		}

		score := toFloat(data["score"])
		total := toFloat(data["totalQuestions"])
		if total > 0 {
			postScoreSum += (score / total) * 100
			postScoreCount++
		}
	}

	pretestScore := 0.0
	if preScoreCount > 0 {
		pretestScore = preScoreSum / preScoreCount
	}

	posttestScore := 0.0
	if postScoreCount > 0 {
		posttestScore = postScoreSum / postScoreCount
	}

	categoryCompleted := map[string]int{
		"linear":    0,
		"trees":     0,
		"graph":     0,
		"sorting":   0,
		"searching": 0,
	}

	for algo, category := range algorithmCategoryCatalog {
		if preByAlgo[algo] && postByAlgo[algo] {
			categoryCompleted[category]++
		}
	}

	categoryProgress := models.UserCategoryAlgoProgressInProfile{
		UserID:    uid,
		Linear:    categoryCompleted["linear"],
		Trees:     categoryCompleted["trees"],
		Graph:     categoryCompleted["graph"],
		Sorting:   categoryCompleted["sorting"],
		Searching: categoryCompleted["searching"],
	}

	totalAlgorithms := len(algorithmCategoryCatalog)
	totalCompleted := 0
	for algo := range algorithmCategoryCatalog {
		if preByAlgo[algo] && postByAlgo[algo] {
			totalCompleted++
		}
	}

	totalProgress := 0.0
	if totalAlgorithms > 0 {
		totalProgress = (float64(totalCompleted) / float64(totalAlgorithms)) * 100
	}

	progress := models.UserProgress{
		UserID:        uid,
		TotalProgress: round2(totalProgress),
		PretestScore:  round2(pretestScore),
		PosttestScore: round2(posttestScore),
	}

	_, err = config.Firestore.Collection("users").Doc(uid).Set(ctx, map[string]interface{}{
		"progress":             progress,
		"categoryAlgoProgress": categoryProgress,
	}, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("failed to update user profile progress: %v", err)
	}

	return nil
}
