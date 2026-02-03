package position

import (
	"strings"
)

// Represents a Connect Four position compactly as a bitboard.
//
// The standard, 6x7 Connect Four board can be represented unambiguously using 49 bits
// in the following bit order:
//
// ```comment
//   6 13 20 27 34 41 48
//  ---------------------
// | 5 12 19 26 33 40 47 |
// | 4 11 18 25 32 39 46 |
// | 3 10 17 24 31 38 45 |
// | 2  9 16 23 30 37 44 |
// | 1  8 15 22 29 36 43 |
// | 0  7 14 21 28 35 42 |
//  ---------------------
//```
//
// The extra row of bits at the top identifies full columns and prevents bits from overflowing
// into the next column. For computational efficiency, positions are stored in practice using two
// `uint64` numbers: one to store a mask of all occupied tiles, and the other to store a mask of the
// current player's tiles.

const (
	W         int = 7
	H         int = 6
	BoardSize int = W * H
	Centre    int = W / 2
	MinScore  int = -(BoardSize)/2 + 3
	MaxScore  int = (BoardSize+1)/2 + 3
)

type Position struct {
	Board uint64
	Mask  uint64
	moves int
}

// a mask for the bottom row of the board.
func bottom_mask() uint64 {
	var mask uint64 = 0
	for i := 0; i < W; i++ {
		mask |= bottom_mask_col(i)
	}
	return mask
}

// A mask for all positions excluding the extra overflow row.
func board_mask() uint64 {
	return bottom_mask() * ((1 << H) - 1)
}

// Creates a new `Position` instance for the initial state of the game.
func NewPosition() *Position {
	p := &Position{
		Board: 0,
		Mask:  0,
		moves: 0,
	}
	return p
}

// Parses a `Position` from a string representation of a Connect Four board.
//
// The input string should contain exactly 42 character from the set ['.', 'o', 'x'],
// representing the board row by row from the top-left to the bottom-right. All other characters
// are ignored. 'x' is the current player, and 'o' is the opponent.
// This method assumes that a correctlt formatted board string is a valid game position.
// Invalid positions will lead to undefined behaviour.
//
// # Arguments
//
// * `board_string`: A string slice representing the board state.
//
// # Returns
//
// On success, returns a `Result` containing the parsed `Position`.
//
// # Errors
//
// Returns a `Error()` if the input string is invalid.

func PositionFromBoardString(board_string string) (*Position, error) {
	board_string = strings.ToLower(board_string)
	var chars []rune
	for _, c := range board_string {
		if c == '.' || c == 'o' || c == 'x' {
			chars = append(chars, c)
		}
	}

	// Validates exact number of `chars` required for a full board
	if len(chars) != BoardSize {
		return nil, InvalidBoardStringLength{Actual: len(chars), Expected: BoardSize}
	}

	var board uint64 = 0
	var mask uint64 = 0
	var moves int = 0

	for i, c := range chars {
		if c == '.' {
			continue
		}

		var row int = H - (i / W) - 1
		var col int = i % W

		var bit_index int = row + col*(H+1)
		var board_bit uint64
		if c == 'x' {
			board_bit = 1
		} else {
			board_bit = 0
		}

		board |= board_bit << bit_index
		mask |= uint64(1) << uint64(bit_index)
		moves += 1
	}

	return &Position{board, mask, moves}, nil
}

func PositionFromMoves(move_sequence string) (*Position, error) {
	var position *Position = NewPosition()
	var col int = -1

	for i, c := range move_sequence {
		if c >= '0' && c <= '9' {
			col = int(c - '0')
		} else {
			return nil, InvalidCharacter{Character: c, Index: i}
		}
		if !position.IsPlayable(col) {
			return nil, InvalidFullColumnMove{Column: col + 1, Index: i}
		}
		if position.IsWinningMove(col) {
			return nil, InvalidWinningMove{Column: col}
		}
		position.Play(col)
	}
	if col == -1 {
		return nil, InvalidColumn{Column: col}
	}
	return position, nil
}

func (self *Position) GetMoves() int {
	return self.moves
}

func (self *Position) GetKey() uint64 {
	// Calculates the standard key for a position
	key := self.Board + self.Mask

	// Calculates the key of the mirrored position
	mirrored_position, mirrored_mask := self.get_mirrored_bitmasks()
	mirrored_key := mirrored_position + mirrored_mask

	if mirrored_key < key {
		return mirrored_key
	}
	return key
}

func (self *Position) get_mirrored_bitmasks() (uint64, uint64) {
	var mirrored_position uint64 = 0
	var mirrored_mask uint64 = 0

	// Swaps columns within the position and mask up to the centre column
	for col := 0; col < Centre; col++ {
		mirrored_col := W - 1 - col
		shift := (mirrored_col - col) * (H + 1)
		mirrored_position |= ((self.Board & column_mask(col)) << uint64(shift)) |
			((self.Board & column_mask(mirrored_col)) >> uint64(shift))
		mirrored_mask |= ((self.Mask & column_mask(col)) << uint64(shift)) |
			((self.Mask & column_mask(mirrored_col)) >> uint64(shift))
	}

	if W&1 == 1 {
		mirrored_position |= self.Board & column_mask(Centre)
		mirrored_mask |= self.Mask & column_mask(Centre)
	}

	return mirrored_position, mirrored_mask
}

// Indicates whether a given column is playable
//
// # Arguments
// `col`: 0-based index of a column
//
// # Returns
//
// True if the column is playable, false if the column is already full
func (self *Position) IsPlayable(col int) bool {
	return self.Mask&top_mask_col(col) != 0
}

// Indicates whether the current player can win with their next move.
// # Arguments
// `col`: 0-based index of a playable column
//
// # Returns
//
// True if the current player make a 4-alignment by playing the column, false if not
func (self *Position) IsWinningMove(col int) bool {
	return self.winning_positions()&self.Possible()&column_mask(col) > 0
}

// Indicates if the current player can win on their next turn
func (self *Position) CanWinNext() bool {
	return self.winning_positions()&self.Possible() > 0
}

// Plays a move in the given column
//
// # Arguments
// `col`: 0-based index of a playable column#
func (self *Position) Play(col int) {
	// Switches the bits of the current and opponent player
	self.Board ^= self.Mask

	// Adds an extra mask bit to the played column
	self.Mask |= self.Mask + bottom_mask_col(col)

	self.moves += 1
}

// Returns a mask for the positionsible moves the current player can make
func (self *Position) Possible() uint64 {
	return self.Mask + bottom_mask()&board_mask()
}

// Returns a mask for the positionsible non losing moves the current player can make
func (self *Position) PossibleNonLosingMoves() uint64 {
	possible := self.Possible()
	opponent_wins := self.opponent_winning_position()

	// Checks if there are any forced moves to avoid the opponent winning
	forced_moves := possible & opponent_wins
	if forced_moves > 0 {
		if forced_moves&(forced_moves-1) > 0 {
			// If the opponent has two winning moves then they can't be stopped
			return 0
		} else {
			possible = forced_moves
		}
	}

	// Avoid playing below any of the opponent's winning positions
	return possible & ^(uint64(opponent_wins) >> 1)
}

func (self *Position) winning_positions() uint64 {
	return compute_winning_position(self.Board, self.Mask)
}

func (self *Position) opponent_winning_position() uint64 {
	return compute_winning_position(self.Board^self.Mask, self.Mask)
}

// Computes a mask for all of a player's winning positions
// Equivalent to a mask of all open ended 3-alignments
// including unreachable floating positions
//
// # Arguments
// * `position`: Bitmask for a player's occupied positions.
// * `mask`: Bitmask for all occupied positions.
//
// # Returns
//
// A bitmask with ones in all positions that a piece could be played by the player to win
func compute_winning_position(position uint64, mask uint64) uint64 {
	// Vertical alignment
	var r uint64 = (position << 1) & (position << 2) & (position << 3)

	// Horizontal alignment
	var p uint64 = (position << (H + 1)) & (position << (2 * (H + 1)))
	r |= p & (position << (3 * (H + 1)))
	r |= p & (position >> (H + 1))
	p >>= 3 * (H + 1)
	r |= p & (position << (H + 1))
	r |= p & (position >> (3 * (H + 1)))

	// Diag alignment 1
	var p2 uint64 = (position << H) & (position << (2 * H))
	r |= p2 & (position << (3 * H))
	r |= p2 & (position >> H)
	p2 >>= 3 * H
	r |= p2 & (position << H)
	r |= p2 & (position >> (3 * H))

	// Diagonal alignment 2
	var p3 uint64 = (position << (H + 2)) & (position << (2 * (H + 2)))
	r |= p3 & (position << (3 * (H + 2)))
	r |= p3 & (position >> (H + 2))
	p3 >>= 3 * (H + 2)
	r |= p3 & (position << (H + 2))
	r |= p3 & (position >> (3 * (H + 2)))

	return r & (board_mask() ^ mask)
}

func (self *Position) ScoreMove(move_bit uint64) uint8 {
	return count_ones(compute_winning_position(self.Board|move_bit, self.Mask))
}

func count_ones(mask uint64) uint8 {
	var count uint8 = 0
	for mask != 0 {
		mask &= mask - 1
		count++
	}
	return count
}

func (self *Position) IsWonPosition() bool {
	return compute_won_position(self.Board) || compute_won_position(self.Board^self.Mask)
}

func compute_won_position(position uint64) bool {
	// Horizontal alignment
	var m uint64 = position & (position >> (H + 1))
	if m&(m>>(2*(H+1))) > 0 {
		return true
	}

	// Diagonal alignment 1
	var m2 uint64 = position & (position >> H)
	if m2&(m2>>(2*H)) > 0 {
		return true
	}

	// Diagonal alignment 2
	var m3 uint64 = position & (position >> (H + 2))
	if m3&(m3>>(2*(H+2))) > 0 {
		return true
	}

	// Vertical alignment
	var m4 uint64 = position & (position >> 1)
	if m4&(m4>>2) > 0 {
		return true
	}
	return false
}

func top_mask_col(col int) uint64 {
	return uint64(1) << (H - 1 + col*(H+1))
}

func bottom_mask_col(col int) uint64 {
	return uint64(1) << (col * (H + 1))
}

func column_mask(col int) uint64 {
	return ((uint64(1) << H) - 1) << (col * (H + 1))
}
