// Package rpsplay implements the Rock-Paper-Scissors "play layer": the pure game
// rules, a stateless game Snapshot that round-trips through Telegram callback
// data, and helpers that render the two game screens as botkb keyboards.
//
// The package keeps NO server-side state. The entire running game (opponent
// mode, score, and the human's per-move history used by the AI predictor) is
// carried in the callback data via Snapshot.Encode / DecodeSnapshot.
package rpsplay

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/bots-go-framework/bots-go-core/botkb"
)

// Move is a single Rock-Paper-Scissors move, stored as its callback-safe byte.
type Move byte

// The three legal moves.
const (
	Rock     Move = 'r'
	Paper    Move = 'p'
	Scissors Move = 's'
)

// Opponent selects how the bot picks its move.
type Opponent string

// The supported opponent modes.
const (
	// OpponentAI predicts the human's most-frequent move and plays what beats it.
	OpponentAI Opponent = "a"
	// OpponentRandom plays a uniform random move.
	OpponentRandom Opponent = "r"
)

// Outcome is the result of a round from the human's perspective.
type Outcome int8

// The three possible outcomes.
const (
	Loss Outcome = -1
	Draw Outcome = 0
	Win  Outcome = 1
)

// Beats reports whether move a beats move b
// (Rock beats Scissors, Scissors beats Paper, Paper beats Rock).
func Beats(a, b Move) bool {
	return (a == Rock && b == Scissors) ||
		(a == Scissors && b == Paper) ||
		(a == Paper && b == Rock)
}

// beatingMove returns the move that beats m.
func beatingMove(m Move) Move {
	switch m {
	case Rock:
		return Paper
	case Paper:
		return Scissors
	case Scissors:
		return Rock
	default:
		return Rock
	}
}

// legalMoves lists the three moves in a stable order.
var legalMoves = [3]Move{Rock, Paper, Scissors}

// randomMove returns a uniformly random legal move.
// A nil rnd uses the math/rand global source.
func randomMove(rnd *rand.Rand) Move {
	var i int
	if rnd == nil {
		i = rand.Intn(len(legalMoves))
	} else {
		i = rnd.Intn(len(legalMoves))
	}
	return legalMoves[i]
}

// Snapshot is the running game state carried in Telegram callback data: the
// opponent mode, the running score (from the human's perspective), and the
// human's per-move counts (used by the AI predictor). No server-side storage.
type Snapshot struct {
	Opponent                Opponent
	Wins, Draws, Losses     int
	Rocks, Papers, Scissors int // how many times the human has played each move
}

// NewGame returns a fresh, zero-score Snapshot for the given opponent.
func NewGame(opp Opponent) Snapshot {
	return Snapshot{Opponent: opp}
}

// Rounds returns the number of rounds played so far (Wins+Draws+Losses).
func (s Snapshot) Rounds() int {
	return s.Wins + s.Draws + s.Losses
}

// predictHuman guesses the human's next move as the most-frequent move played
// so far. With a unique most-frequent move it returns that move; on a tie (which
// includes an empty history) it returns a uniformly random move.
func (s Snapshot) predictHuman(rnd *rand.Rand) Move {
	r, p, sc := s.Rocks, s.Papers, s.Scissors
	switch {
	case r > p && r > sc:
		return Rock
	case p > r && p > sc:
		return Paper
	case sc > r && sc > p:
		return Scissors
	default:
		return randomMove(rnd)
	}
}

// Play resolves one round: the human plays human; the opponent picks a move per
// its mode; it returns the updated snapshot (score and the human's move counts
// advanced) plus the opponent's move and the outcome from the human's
// perspective. rnd is used by the Random opponent and by the AI opponent's
// tie/empty-history fallback; a nil rnd uses the math/rand global source.
func (s Snapshot) Play(human Move, rnd *rand.Rand) (next Snapshot, botMove Move, outcome Outcome) {
	switch s.Opponent {
	case OpponentAI:
		botMove = beatingMove(s.predictHuman(rnd))
	default: // OpponentRandom and any unknown mode
		botMove = randomMove(rnd)
	}

	switch {
	case human == botMove:
		outcome = Draw
	case Beats(human, botMove):
		outcome = Win
	default:
		outcome = Loss
	}

	next = s
	switch human {
	case Rock:
		next.Rocks++
	case Paper:
		next.Papers++
	case Scissors:
		next.Scissors++
	}
	switch outcome {
	case Win:
		next.Wins++
	case Draw:
		next.Draws++
	case Loss:
		next.Losses++
	}
	return next, botMove, outcome
}

// ErrInvalidSnapshot wraps every DecodeSnapshot failure; match it with errors.Is.
var ErrInvalidSnapshot = errors.New("rpsplay: invalid snapshot")

// Encode serializes the Snapshot to a compact, callback-safe string with no
// spaces: "<opp>.<w>.<d>.<l>.<rocks>.<papers>.<scissors>".
func (s Snapshot) Encode() string {
	return fmt.Sprintf("%s.%d.%d.%d.%d.%d.%d",
		s.Opponent, s.Wins, s.Draws, s.Losses, s.Rocks, s.Papers, s.Scissors)
}

// DecodeSnapshot is the exact inverse of Snapshot.Encode. It validates the
// opponent character and parses the six integer fields, wrapping every failure
// in ErrInvalidSnapshot.
func DecodeSnapshot(encoded string) (Snapshot, error) {
	parts := strings.Split(encoded, ".")
	if len(parts) != 7 {
		return Snapshot{}, fmt.Errorf("%w: expected 7 fields, got %d", ErrInvalidSnapshot, len(parts))
	}

	opp := Opponent(parts[0])
	switch opp {
	case OpponentAI, OpponentRandom:
		// ok
	default:
		return Snapshot{}, fmt.Errorf("%w: unknown opponent %q", ErrInvalidSnapshot, parts[0])
	}

	nums := make([]int, 6)
	for i := 0; i < 6; i++ {
		v, err := strconv.Atoi(parts[i+1])
		if err != nil {
			return Snapshot{}, fmt.Errorf("%w: field %d: %v", ErrInvalidSnapshot, i+1, err)
		}
		nums[i] = v
	}

	return Snapshot{
		Opponent: opp,
		Wins:     nums[0],
		Draws:    nums[1],
		Losses:   nums[2],
		Rocks:    nums[3],
		Papers:   nums[4],
		Scissors: nums[5],
	}, nil
}

// glyph returns the emoji for a move.
func glyph(m Move) string {
	switch m {
	case Rock:
		return "🪨"
	case Paper:
		return "📄"
	case Scissors:
		return "✂️"
	default:
		return "?"
	}
}

// scoreLine formats the running score, e.g. "Score: 3–1–2 (W–D–L)".
func scoreLine(s Snapshot) string {
	return fmt.Sprintf("Score: %d–%d–%d (W–D–L)", s.Wins, s.Draws, s.Losses)
}

// Rendered is a screen: message text plus the keyboard to attach to it.
type Rendered struct {
	Text     string
	Keyboard botkb.Keyboard
}

// RenderChoose renders the "pick your move" screen: a status line with the
// running score, then a row of three move buttons (🪨 Rock / 📄 Paper /
// ✂️ Scissors) whose callback data comes from moveData(move), plus a Back
// button carrying backData.
func RenderChoose(s Snapshot, moveData func(m Move) string, backData string) Rendered {
	text := "Pick your move\n" + scoreLine(s)
	moveRow := []botkb.Button{
		botkb.NewDataButton(glyph(Rock)+" Rock", moveData(Rock)),
		botkb.NewDataButton(glyph(Paper)+" Paper", moveData(Paper)),
		botkb.NewDataButton(glyph(Scissors)+" Scissors", moveData(Scissors)),
	}
	backRow := []botkb.Button{botkb.NewDataButton("⬅️ Back", backData)}
	kb := botkb.NewMessageKeyboard(botkb.KeyboardTypeInline, moveRow, backRow)
	return Rendered{Text: text, Keyboard: kb}
}

// RenderResult renders the outcome of a round: "You: <glyph>  Bot: <glyph>", the
// result (✅ You win! / 🤝 Draw / ❌ You lose), the running score, then a
// "🔁 Play again" button (callback playAgainData) and a Back button (backData).
func RenderResult(s Snapshot, human, bot Move, outcome Outcome, playAgainData, backData string) Rendered {
	var result string
	switch outcome {
	case Win:
		result = "✅ You win!"
	case Draw:
		result = "🤝 Draw"
	default: // Loss
		result = "❌ You lose"
	}
	text := fmt.Sprintf("You: %s  Bot: %s\n%s\n%s", glyph(human), glyph(bot), result, scoreLine(s))
	playRow := []botkb.Button{botkb.NewDataButton("🔁 Play again", playAgainData)}
	backRow := []botkb.Button{botkb.NewDataButton("⬅️ Back", backData)}
	kb := botkb.NewMessageKeyboard(botkb.KeyboardTypeInline, playRow, backRow)
	return Rendered{Text: text, Keyboard: kb}
}
