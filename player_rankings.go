package main

import "math"

func getExpectedScore(a int, b int) float64 {
	ex := (float64(b) - float64(a)) / 400.0

	return 1 / (1 + math.Pow(10.0, ex))
}

// winner - -1, b won. 0 draw, 1 a won.
// returns difference for a
func getEloDifference(a int, b int, winner int, k float32) int {

	aexpected := getExpectedScore(a, b)
	score := 0.0
	if winner == 1 {
		score = 1
	} else if winner == 0 {
		score = 0.5
	}
	scale := float32(score) - float32(aexpected)
	return int(scale * k)
}

// winner - if a won, pass 1. If b won, pass -1. If draw, pass 0
func GetELoAdjustments(a int, b int, winner int, k float32) (int, int) {
	return getEloDifference(a, b, winner, k), getEloDifference(b, a, -winner, k)
}
