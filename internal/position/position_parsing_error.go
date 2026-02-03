package position

import "fmt"

type InvalidBoardStringLength struct {
	Actual   int
	Expected int
}

type InvalidCharacter struct {
	Character rune
	Index     int
}

type InvalidColumn struct {
	Column int
	Index  int
}

type InvalidFullColumnMove struct {
	Column int
	Index  int
}

type InvalidWinningMove struct {
	Column int
	Index  int
}

func (e InvalidBoardStringLength) Error() string {
	return fmt.Sprintf("invalid board string length: found %d, expected %d", e.Actual, e.Expected)
}

func (e InvalidCharacter) Error() string {
	return fmt.Sprintf("invalid character: character '%c' at index %d", e.Character, e.Index)
}

func (e InvalidColumn) Error() string {
	return fmt.Sprintf("invalid column %d at index %d", e.Column, e.Index)
}

func (e InvalidFullColumnMove) Error() string {
	return fmt.Sprintf("invalid move at index %d: column %d is full", e.Index, e.Column)
}

func (e InvalidWinningMove) Error() string {
	return fmt.Sprintf("invalid move at index %d: column %d results in a win", e.Index, e.Column)
}
