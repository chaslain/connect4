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
	if column > 8 {
		return false
	}
	col := &board.Columns[column]

	piecesPlayed := len(col.Rows)

	if piecesPlayed == 8 {
		return false
	}

	col.Rows = append(col.Rows, player)
	return true
}
