---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Telegram Rock-Paper-Scissors bot (vs AI/Random, running score)

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/sneat-games/rock-paper-scissors/spec/features/telegram-rps-bot?op=explore) | [Edit](https://specscore.studio/app/github.com/sneat-games/rock-paper-scissors/spec/features/telegram-rps-bot?op=edit) | [Ask question](https://specscore.studio/app/github.com/sneat-games/rock-paper-scissors/spec/features/telegram-rps-bot?op=ask) | [Request change](https://specscore.studio/app/github.com/sneat-games/rock-paper-scissors/spec/features/telegram-rps-bot?op=request-change) |
**Status:** Draft
**Source Ideas:** rps-as-a-sneat-bot-game

## Summary

Play **Rock-Paper-Scissors** against a built-in opponent — a **pattern-predicting
AI** or **Random** — inside a Telegram chat, keeping a **running win–draw–loss
score**. The whole game state (opponent mode, score, and the human's per-move
counts for the AI) travels in the button **callback data**, so there is no
server-side storage. This is the second game in SneatBot's `/games` (after
[Reversi](https://github.com/sneat-games/reversi/blob/main/spec/features/telegram-reversi-bot/README.md)),
built to the same pattern: a framework-light play layer (`rpsplay`) driven by any
Sneat bot.

This document follows the [SpecScore feature specification](https://specscore.md/feature-specification).

## Problem

The legacy Rock-Paper-Scissors app here was a standalone Google App Engine bot on
`strongo/bots-framework` oriented around multiplayer tournaments and betting — and
it never had real game logic (`rpsfacade.MakeMove` was an empty stub). Meanwhile
SneatBot's `/games` menu wants a second, fast, casual game after Reversi. RPS is
ideal: instant rounds, trivial rules, no board, and no persistence needed. The one
design risk is that a naive RPS opponent (pure random) can feel pointless — so the
default AI adds a light pattern-predicting twist to keep it engaging.

## Behavior

### Opponent modes

#### REQ: opponent-modes

The human MUST be able to play against one of two opponents, chosen when starting
a game:

- **AI** — predicts the human's **most-frequently-played move so far** and plays
  the move that beats that prediction. On a tie, or with no history yet, it plays
  a uniformly random move.
- **Random** — plays a uniformly random move.

The chosen mode MUST persist for the whole session so every round uses the same
opponent.

### Playing a round

#### REQ: rules

Rock beats Scissors, Scissors beats Paper, Paper beats Rock; equal moves draw.
Rules MUST come from the `rpsplay` engine (`Beats`/`Play`); the bot layer MUST NOT
re-implement them.

#### REQ: choose-and-resolve

The game MUST present the three moves (🪨 Rock, 📄 Paper, ✂️ Scissors) as buttons.
Tapping one MUST resolve the round: the opponent picks per its mode, and the result
screen MUST show the human's move, the opponent's move, and the outcome (win / draw
/ loss from the human's perspective).

#### REQ: running-score

A running **win–draw–loss** score (from the human's perspective) MUST be kept
across rounds and shown on the choose and result screens. "🔁 Play again" MUST
continue the same session, preserving the score and opponent.

### State model

#### REQ: state-in-callback-data

The complete game state — opponent mode, the win/draw/loss score, and the human's
per-move counts (rock/paper/scissors, used by the AI predictor) — MUST be encoded
in each button's `callback_data`, within Telegram's 64-byte limit, and MUST NOT be
persisted server-side. The snapshot MUST round-trip exactly through encode/decode.

#### REQ: current-bots-framework

The bot integration MUST target the current `bots-go-framework` modules (`bots-fw`,
`bots-go-core/botkb`, `bots-fw-telegram`), not the legacy `strongo/bots-framework`.
Rendered keyboards MUST be built with `botkb`.

## Architecture & Components

- **`rpsplay` play layer** (this repo, `server-go/rpsplay`) — the engine
  (`Move`, `Beats`, `Play`, `Outcome`), the two opponents (AI predictor + Random),
  the `Snapshot` (opponent + score + move counts) with `Encode`/`DecodeSnapshot`,
  and `RenderChoose`/`RenderResult` returning a `botkb` keyboard + text. Depends
  only on the standard library + `bots-go-core/botkb`; the host supplies each
  button's `callback_data` via the render params.
- **Host-bot wiring** (out of this repo — SneatBot in `sneat-co/sneat-go`)
  registers the commands, calls `rpsplay`, and maps the keyboard to Telegram.

**Callback-data layout** (illustrative): `<opp>.<w>.<d>.<l>.<rocks>.<papers>.<scissors>`
— the opponent char plus six small integers; well under 64 bytes even after the
host adds a command prefix + the chosen move.

## Not Doing / Out of Scope (v1)

- Player-vs-player (two humans), invites, matchmaking, tournaments, betting (the
  legacy app's focus).
- Server-side history / leaderboards / persistence beyond the current snapshot.
- A stronger AI than the most-frequent-move predictor.
- Other messengers — only Telegram (via SneatBot) for now.

## Acceptance Criteria

### AC: opponent-modes-selectable (verifies REQ:opponent-modes)

**Given** the human chooses the AI opponent (and separately, Random)
**When** a round is played
**Then** Random returns a uniformly-random legal move, and AI returns the move that
beats the human's most-frequently-played move (a random legal move when there is a
tie or no history).

### AC: rules-from-engine (verifies REQ:rules)

**Given** two moves
**When** the winner is determined
**Then** it follows Rock>Scissors, Scissors>Paper, Paper>Rock (equal = draw), computed
by the `rpsplay` engine, with no rule logic duplicated in the bot layer.

### AC: round-shows-both-moves-and-outcome (verifies REQ:choose-and-resolve)

**Given** the choose screen with 🪨/📄/✂️ buttons
**When** the human taps a move
**Then** the result screen shows the human's move, the opponent's move, and the
outcome (win/draw/loss).

### AC: score-accumulates (verifies REQ:running-score)

**Given** a session with some rounds played
**When** another round resolves and "Play again" is tapped
**Then** the win/draw/loss score reflects all rounds so far and is shown, and the
same opponent continues.

### AC: state-round-trips-within-limit (verifies REQ:state-in-callback-data)

**Given** any reachable snapshot (opponent + score + move counts)
**When** it is encoded into a button's `callback_data` and decoded
**Then** the decoded snapshot equals the original and the encoded `callback_data`
is at most 64 bytes, with nothing written to any server-side store.

### AC: uses-current-framework (verifies REQ:current-bots-framework)

**Given** the built RPS bot integration and its module graph
**When** its dependencies are inspected
**Then** it imports `bots-go-framework` modules (`bots-fw`, `bots-go-core/botkb`,
`bots-fw-telegram`) and not `github.com/strongo/bots-framework`.

## Open Questions

None at this time.

---
*This document follows the https://specscore.md/feature-specification*
