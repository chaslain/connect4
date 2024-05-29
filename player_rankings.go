package main

func getExpectedScore(a int, b int) int {
	ex := (b - a) / 400

	return 1 / (1 + 10 ^ ex)
}

// winner - -1, b won. 0 draw, 1 a won.
// returns difference for a
func getEloDifference(a int, b int, winner int, k float32) float32 {

	aexpected := getExpectedScore(a, b)
	actual := float32(winner) * 0.5
	scale := float32(aexpected) - float32(actual)
	return scale * k
}

// winner - if a won, pass 1. If b won, pass -1. If draw, pass 0
func GetELoAdjustments(a int, b int, winner int, k float32) (float32, float32) {
	return getEloDifference(a, b, winner, k), getEloDifference(b, a, winner*-1, k)
}
