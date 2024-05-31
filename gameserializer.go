package main

import (
	"encoding/json"
	"log"
)

type Board struct {
	Columns []Column
}

type Column struct {
	Rows []int
}

func EmptyBoard() Board {
	return Board{
		Columns: []Column{
			{
				Rows: []int{},
			},
			{
				Rows: []int{},
			},
			{
				Rows: []int{},
			},
			{
				Rows: []int{},
			},
			{
				Rows: []int{},
			},
			{
				Rows: []int{},
			},
			{
				Rows: []int{},
			},
			{
				Rows: []int{},
			},
			{
				Rows: []int{},
			},
		},
	}
}

func GetGame(serialized string) Board {
	var result Board
	json.Unmarshal([]byte(serialized), &result)
	return result
}

func GetSerial(board Board) string {
	d, e := json.Marshal(board)
	if e != nil {
		log.Default().Println("Failed to serialize game ")
	}

	return string(d)
}

func PlayMove(board *Board, column int, player int) bool {
	if column > 7 {
		return false
	}
	col := &board.Columns[column]

	piecesPlayed := len(col.Rows)

	if piecesPlayed == 6 {
		return false
	}

	col.Rows = append(col.Rows, player)
	return true
}

func CheckForWin(board *Board, column int, player int) bool {
	row := len(board.Columns[column].Rows) - 1
	target := board.Columns[column].Rows[row]

	return checkDiagnoalDescending(board, column, row, target) ||
		checkHorizontal(board, row, target) ||
		checkDiagnoalAscending(board, column, row, target) ||
		checkVertical(board, column, row, target)
}

func checkHorizontal(board *Board, row int, target int) bool {
	count := 0
	for i := 0; i < 8; i++ {
		if len(board.Columns[i].Rows) <= row {
			count = 0
			continue
		}
		if target == board.Columns[i].Rows[row] {
			count++
			if count > 3 {
				return true
			}
		} else {
			count = 0
		}
	}

	return false
}

func checkDiagnoalDescending(board *Board, column int, row int, target int) bool {
	count := 0

	for column > 0 && row < 5 {
		column--
		row++
	}

	for column < 7 && row >= 0 {
		if len(board.Columns[column].Rows) <= row {
			column++
			row--
			count = 0
			continue
		}
		if board.Columns[column].Rows[row] == target {
			count++
			if count > 3 {
				return true
			}
		} else {
			count = 0
		}
		column++
		row--
	}

	return false
}

func checkDiagnoalAscending(board *Board, column int, row int, target int) bool {
	count := 0

	// go the edge of the board
	for row > 0 && column > 0 {
		row--
		column--
	}

	for column < 8 && row < 7 {
		if len(board.Columns[column].Rows) <= row {
			column++
			row++
			count = 0
			continue
		}
		if board.Columns[column].Rows[row] == target {
			count++
			if count > 3 {
				return true
			}
		} else {
			count = 0
		}
		column++
		row++
	}

	return false
}

func checkVertical(board *Board, column int, row int, target int) bool {
	count := 0
	for row >= 0 {
		if target != board.Columns[column].Rows[row] {
			return false
		} else {
			count++
			if count > 3 {
				return true
			}
		}
		row--
	}

	return false
}
