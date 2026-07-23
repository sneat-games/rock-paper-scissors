package rpsplay

import (
	"errors"
	"math/rand"
	"strings"
	"testing"

	"github.com/bots-go-framework/bots-go-core/botkb"
)

// isLegalMove reports whether m is one of the three legal moves.
func isLegalMove(m Move) bool {
	return m == Rock || m == Paper || m == Scissors
}

func TestBeats(t *testing.T) {
	wins := []struct{ a, b Move }{
		{Rock, Scissors},
		{Scissors, Paper},
		{Paper, Rock},
	}
	for _, w := range wins {
		if !Beats(w.a, w.b) {
			t.Errorf("Beats(%c, %c) = false, want true", w.a, w.b)
		}
		// The reverse pairing must lose.
		if Beats(w.b, w.a) {
			t.Errorf("Beats(%c, %c) = true, want false (reverse)", w.b, w.a)
		}
	}
	// Equal pairs never beat.
	for _, m := range []Move{Rock, Paper, Scissors} {
		if Beats(m, m) {
			t.Errorf("Beats(%c, %c) = true, want false (equal)", m, m)
		}
	}
}

func TestNewGameAndRounds(t *testing.T) {
	s := NewGame(OpponentAI)
	if s.Opponent != OpponentAI {
		t.Errorf("Opponent = %q, want %q", s.Opponent, OpponentAI)
	}
	if s.Wins != 0 || s.Draws != 0 || s.Losses != 0 {
		t.Errorf("fresh score = %d/%d/%d, want 0/0/0", s.Wins, s.Draws, s.Losses)
	}
	if s.Rocks != 0 || s.Papers != 0 || s.Scissors != 0 {
		t.Errorf("fresh counts = %d/%d/%d, want 0/0/0", s.Rocks, s.Papers, s.Scissors)
	}
	if got := s.Rounds(); got != 0 {
		t.Errorf("fresh Rounds() = %d, want 0", got)
	}

	s2 := Snapshot{Wins: 3, Draws: 1, Losses: 2}
	if got := s2.Rounds(); got != 6 {
		t.Errorf("Rounds() = %d, want 6", got)
	}
}

func TestPlayOutcomes(t *testing.T) {
	// Fixed opponent moves are achieved via the Random opponent seeded so that
	// we assert only on the outcome logic given the returned botMove.
	// Here we test the pure outcome mapping through Play by driving the AI
	// opponent with a known prediction, but simplest is to test outcome logic
	// directly against every (human, bot) pair.
	cases := []struct {
		human, bot Move
		want       Outcome
	}{
		{Rock, Scissors, Win},
		{Scissors, Paper, Win},
		{Paper, Rock, Win},
		{Rock, Rock, Draw},
		{Paper, Paper, Draw},
		{Scissors, Scissors, Draw},
		{Scissors, Rock, Loss},
		{Paper, Scissors, Loss},
		{Rock, Paper, Loss},
	}

	for _, c := range cases {
		// Build an AI snapshot whose history forces a known prediction so the
		// bot move is deterministic: the AI plays beatingMove(prediction).
		// We instead verify the outcome+increments via a random opponent with a
		// seed that yields c.bot, but the cleanest deterministic path is to
		// check the exported outcome mapping using a helper below.
		got := outcomeOf(c.human, c.bot)
		if got != c.want {
			t.Errorf("outcomeOf(%c, %c) = %d, want %d", c.human, c.bot, got, c.want)
		}
	}

	// Now assert Play advances the right score field and the human move count,
	// and returns a legal bot move. Use a random opponent with a fixed seed.
	rnd := rand.New(rand.NewSource(1))
	start := NewGame(OpponentRandom)
	next, botMove, outcome := start.Play(Rock, rnd)
	if !isLegalMove(botMove) {
		t.Errorf("Play botMove = %c, not a legal move", botMove)
	}
	if next.Rocks != 1 {
		t.Errorf("after Play(Rock) Rocks = %d, want 1", next.Rocks)
	}
	// Exactly one score field must have advanced by one, matching the outcome.
	switch outcome {
	case Win:
		if next.Wins != 1 || next.Draws != 0 || next.Losses != 0 {
			t.Errorf("Win outcome score = %d/%d/%d, want 1/0/0", next.Wins, next.Draws, next.Losses)
		}
	case Draw:
		if next.Wins != 0 || next.Draws != 1 || next.Losses != 0 {
			t.Errorf("Draw outcome score = %d/%d/%d, want 0/1/0", next.Wins, next.Draws, next.Losses)
		}
	case Loss:
		if next.Wins != 0 || next.Draws != 0 || next.Losses != 1 {
			t.Errorf("Loss outcome score = %d/%d/%d, want 0/0/1", next.Wins, next.Draws, next.Losses)
		}
	}
	if next.Rounds() != 1 {
		t.Errorf("after one Play Rounds() = %d, want 1", next.Rounds())
	}
}

// outcomeOf mirrors Play's outcome mapping for direct assertion.
func outcomeOf(human, bot Move) Outcome {
	switch {
	case human == bot:
		return Draw
	case Beats(human, bot):
		return Win
	default:
		return Loss
	}
}

func TestPlayScoreAndCountIncrementForEachMove(t *testing.T) {
	rnd := rand.New(rand.NewSource(42))
	for _, human := range []Move{Rock, Paper, Scissors} {
		s := NewGame(OpponentRandom)
		next, _, _ := s.Play(human, rnd)
		switch human {
		case Rock:
			if next.Rocks != 1 || next.Papers != 0 || next.Scissors != 0 {
				t.Errorf("Play(Rock) counts = %d/%d/%d, want 1/0/0", next.Rocks, next.Papers, next.Scissors)
			}
		case Paper:
			if next.Papers != 1 || next.Rocks != 0 || next.Scissors != 0 {
				t.Errorf("Play(Paper) counts = %d/%d/%d, want 0/1/0", next.Rocks, next.Papers, next.Scissors)
			}
		case Scissors:
			if next.Scissors != 1 || next.Rocks != 0 || next.Papers != 0 {
				t.Errorf("Play(Scissors) counts = %d/%d/%d, want 0/0/1", next.Rocks, next.Papers, next.Scissors)
			}
		}
	}
}

func TestPlayRandomOnlyLegalMoves(t *testing.T) {
	rnd := rand.New(rand.NewSource(7))
	s := NewGame(OpponentRandom)
	for i := 0; i < 5000; i++ {
		next, botMove, _ := s.Play(Rock, rnd)
		if !isLegalMove(botMove) {
			t.Fatalf("iteration %d: illegal bot move %c", i, botMove)
		}
		s = next
	}
}

func TestPlayAICountersMostFrequentMove(t *testing.T) {
	rnd := rand.New(rand.NewSource(3))

	// Human has played Rock several times => AI predicts Rock, plays Paper.
	s := Snapshot{Opponent: OpponentAI, Rocks: 5}
	_, botMove, outcome := s.Play(Rock, rnd)
	if botMove != Paper {
		t.Errorf("AI vs Rock-heavy history: botMove = %c, want Paper (%c)", botMove, Paper)
	}
	if !Beats(botMove, Rock) {
		t.Errorf("AI botMove %c does not beat the predicted Rock", botMove)
	}
	if outcome != Loss {
		t.Errorf("human plays Rock into AI Paper: outcome = %d, want Loss", outcome)
	}

	// Paper-heavy history => predicts Paper, plays Scissors.
	s = Snapshot{Opponent: OpponentAI, Papers: 9}
	_, botMove, _ = s.Play(Paper, rnd)
	if botMove != Scissors {
		t.Errorf("AI vs Paper-heavy history: botMove = %c, want Scissors (%c)", botMove, Scissors)
	}

	// Scissors-heavy history => predicts Scissors, plays Rock.
	s = Snapshot{Opponent: OpponentAI, Scissors: 4}
	_, botMove, _ = s.Play(Scissors, rnd)
	if botMove != Rock {
		t.Errorf("AI vs Scissors-heavy history: botMove = %c, want Rock (%c)", botMove, Rock)
	}

	// No history: AI must still return a legal move (seeded rnd).
	empty := NewGame(OpponentAI)
	for i := 0; i < 1000; i++ {
		_, botMove, _ = empty.Play(Rock, rnd)
		if !isLegalMove(botMove) {
			t.Fatalf("AI with no history returned illegal move %c", botMove)
		}
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	snaps := []Snapshot{
		NewGame(OpponentAI),
		NewGame(OpponentRandom),
		{Opponent: OpponentAI, Wins: 3, Draws: 1, Losses: 2, Rocks: 3, Papers: 2, Scissors: 1},
		{Opponent: OpponentRandom, Wins: 123, Draws: 45, Losses: 67, Rocks: 100, Papers: 55, Scissors: 80},
		{Opponent: OpponentAI, Wins: 12345, Draws: 6789, Losses: 100, Rocks: 9999, Papers: 8888, Scissors: 7777},
	}
	for _, s := range snaps {
		enc := s.Encode()
		if strings.ContainsAny(enc, " \t\n") {
			t.Errorf("Encode(%v) = %q contains whitespace", s, enc)
		}
		got, err := DecodeSnapshot(enc)
		if err != nil {
			t.Errorf("DecodeSnapshot(%q) unexpected error: %v", enc, err)
			continue
		}
		if got != s {
			t.Errorf("round-trip mismatch: got %+v, want %+v (enc %q)", got, s, enc)
		}
	}
}

func TestEncodeLength(t *testing.T) {
	// Even large-but-realistic counts must fit callback-data budget (<= 40).
	snaps := []Snapshot{
		{Opponent: OpponentAI, Wins: 3, Draws: 1, Losses: 2, Rocks: 3, Papers: 2, Scissors: 1},
		{Opponent: OpponentRandom, Wins: 999, Draws: 999, Losses: 999, Rocks: 999, Papers: 999, Scissors: 999},
		{Opponent: OpponentAI, Wins: 99999, Draws: 99999, Losses: 99999, Rocks: 99999, Papers: 99999, Scissors: 99999},
	}
	for _, s := range snaps {
		if got := len(s.Encode()); got > 40 {
			t.Errorf("len(Encode()) = %d for %+v, want <= 40 (enc %q)", got, s, s.Encode())
		}
	}
}

func TestEncodeSample(t *testing.T) {
	s := Snapshot{Opponent: OpponentAI, Wins: 3, Draws: 1, Losses: 2, Rocks: 3, Papers: 2, Scissors: 1}
	if got := s.Encode(); got != "a.3.1.2.3.2.1" {
		t.Errorf("Encode() = %q, want %q", got, "a.3.1.2.3.2.1")
	}
}

func TestDecodeSnapshotErrors(t *testing.T) {
	bad := []string{
		"",                  // empty
		"garbage",           // single field, not enough parts
		"x.0.0.0.0.0.0",     // bad opponent char
		"a.0.0.0.0.0",       // too few fields
		"a.0.0.0.0.0.0.0",   // too many fields
		"a.0.0.0.0.0.z",     // non-numeric field
		"a.1.2.three.4.5.6", // non-numeric field mid-list
		"1.2.3.4.5.6.7",     // numeric opponent, invalid
	}
	for _, in := range bad {
		_, err := DecodeSnapshot(in)
		if err == nil {
			t.Errorf("DecodeSnapshot(%q) = nil error, want ErrInvalidSnapshot", in)
			continue
		}
		if !errors.Is(err, ErrInvalidSnapshot) {
			t.Errorf("DecodeSnapshot(%q) error %v does not match ErrInvalidSnapshot", in, err)
		}
	}
}

// keyboardButtons type-asserts a botkb.Keyboard to its concrete MessageKeyboard
// and returns its rows for inspection.
func keyboardButtons(t *testing.T, kb botkb.Keyboard) [][]botkb.Button {
	t.Helper()
	mk, ok := kb.(*botkb.MessageKeyboard)
	if !ok {
		t.Fatalf("keyboard is %T, want *botkb.MessageKeyboard", kb)
	}
	return mk.Buttons
}

// dataOf type-asserts a button to *botkb.DataButton and returns its callback data.
func dataOf(t *testing.T, b botkb.Button) string {
	t.Helper()
	db, ok := b.(*botkb.DataButton)
	if !ok {
		t.Fatalf("button is %T, want *botkb.DataButton", b)
	}
	return db.Data
}

func TestRenderChoose(t *testing.T) {
	s := Snapshot{Opponent: OpponentAI, Wins: 3, Draws: 1, Losses: 2}
	moveData := func(m Move) string { return "mv:" + string(m) }
	const backData = "back:menu"

	r := RenderChoose(s, moveData, backData)

	if !strings.Contains(r.Text, "3–1–2") {
		t.Errorf("RenderChoose text %q does not mention the score", r.Text)
	}

	rows := keyboardButtons(t, r.Keyboard)
	if len(rows) < 2 {
		t.Fatalf("RenderChoose keyboard has %d rows, want >= 2", len(rows))
	}

	moveRow := rows[0]
	if len(moveRow) != 3 {
		t.Fatalf("move row has %d buttons, want 3", len(moveRow))
	}
	for i, m := range []Move{Rock, Paper, Scissors} {
		if got := dataOf(t, moveRow[i]); got != moveData(m) {
			t.Errorf("move button %d data = %q, want %q", i, got, moveData(m))
		}
	}

	// The Back button must appear somewhere carrying backData.
	if got := dataOf(t, rows[len(rows)-1][0]); got != backData {
		t.Errorf("back button data = %q, want %q", got, backData)
	}
}

func TestRenderResult(t *testing.T) {
	s := Snapshot{Opponent: OpponentAI, Wins: 1, Draws: 0, Losses: 0}
	const playAgain = "again:1"
	const backData = "back:menu"

	r := RenderResult(s, Rock, Scissors, Win, playAgain, backData)

	if !strings.Contains(r.Text, glyph(Rock)) || !strings.Contains(r.Text, glyph(Scissors)) {
		t.Errorf("RenderResult text %q missing one of the move glyphs", r.Text)
	}
	if !strings.Contains(r.Text, "win") && !strings.Contains(r.Text, "You win") {
		t.Errorf("RenderResult text %q does not name the Win outcome", r.Text)
	}

	rows := keyboardButtons(t, r.Keyboard)
	var foundPlayAgain, foundBack bool
	for _, row := range rows {
		for _, b := range row {
			switch dataOf(t, b) {
			case playAgain:
				foundPlayAgain = true
			case backData:
				foundBack = true
			}
		}
	}
	if !foundPlayAgain {
		t.Errorf("RenderResult keyboard missing Play-again button with data %q", playAgain)
	}
	if !foundBack {
		t.Errorf("RenderResult keyboard missing Back button with data %q", backData)
	}

	// Sanity-check the other outcomes render their labels.
	if got := RenderResult(s, Rock, Rock, Draw, playAgain, backData); !strings.Contains(got.Text, "Draw") {
		t.Errorf("Draw result text %q does not name Draw", got.Text)
	}
	if got := RenderResult(s, Rock, Paper, Loss, playAgain, backData); !strings.Contains(got.Text, "lose") {
		t.Errorf("Loss result text %q does not name the loss", got.Text)
	}
}
