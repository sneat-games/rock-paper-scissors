---
format: https://specscore.md/idea-specification
status: Specifying
---

# Idea: Rock-Paper-Scissors as a Sneat bot game

**Status:** Specifying
**Date:** 2026-07-23
**Owner:** alex
**Promotes To:** telegram-rps-bot
**Supersedes:** —
**Related Ideas:** —

## Problem Statement

How might we add a fast, zero-persistence Rock-Paper-Scissors game to Sneat's /games that stays fun via a light pattern-predicting AI?

## Context

SneatBot's `/games` shipped with **Reversi** first, built as a framework-light
play layer in its own repo. The next game should be quick and casual. This RPS
repo already exists (`prizarena/rock-paper-scissors`) but as a dead standalone GAE
app on `strongo/bots-framework` with no real game logic — so RPS is a fresh build,
mirroring the Reversi pattern (game-per-repo, state in callback data). Prior art:
the [Reversi feature](../features/../../../reversi/spec/features/telegram-reversi-bot/README.md)
and the ecosystem [`games`](../../../../sneat-co/backstage/spec/features/games/README.md) feature.

## Recommended Direction

Build a small `rpsplay` play layer (engine + two opponents + snapshot codec +
render), released from this repo and mounted in SneatBot's `/games` just like
Reversi. Keep **all state in the button callback data** — opponent mode, a running
win–draw–loss score, and the human's per-move counts — so there is no persistence.

Offer two opponents: **Random**, and an **AI that predicts the human's
most-frequently-played move and counters it**. The AI is what keeps RPS engaging:
pure random feels pointless, but a light predictor rewards varying your play
without being unbeatable. Rounds are instant, so — unlike Reversi — no "thinking"
delay is needed.

## Alternatives Considered

- **Single round, no score** — lost: a running score across rounds is what creates
  a session worth replaying; it's nearly free to carry in callback data.
- **Pure-random opponent only** — lost: RPS vs pure random has no skill and gets
  boring fast; the most-frequent-move predictor adds just enough challenge.
- **Reuse the legacy RPS app** — lost: it's a dead GAE/tournament/betting app on
  `strongo/bots-framework` with an empty `MakeMove` stub — nothing to reuse.

## MVP Scope

A player can open Rock-Paper-Scissors in a Telegram chat, pick an opponent (AI or
Random), play rounds by tapping 🪨/📄/✂️, see each round's result and a running
win–draw–loss score, and keep playing — with **no state stored anywhere but the
message**.

## Not Doing (and Why)

- Player-vs-player, invites, tournaments, betting — the legacy app's focus; out of scope for a casual solo game.
- Server-side history / leaderboards / persistence — everything lives in callback data.
- A stronger AI than the most-frequent-move predictor — enough for casual play; revisit if it feels weak.
- Other messengers — only Telegram (via SneatBot) for now.

## Key Assumptions to Validate

| Tier | Assumption | How to validate |
|------|------------|-----------------|
| Must-be-true | Opponent + score + per-move counts encode into `callback_data` within Telegram's 64-byte limit for any reachable session. | Round-trip test with large counts; measure encoded length. |
| Should-be-true | The most-frequent-move AI feels more fun than pure random without being unbeatable. | Play-test; confirm it counters repeated moves but stays beatable by mixing. |
| Might-be-true | Players want a running score rather than one-off rounds. | Ship to `/games`; watch replay behaviour. |

## SpecScore Integration

- **New Features this would create:** [`telegram-rps-bot`](../features/telegram-rps-bot/README.md)
- **Existing Features affected:** none
- **Dependencies:** host-bot wiring in `sneat-co/sneat-go` (SneatBot `/games`) and the ecosystem `games` feature in `sneat-co/backstage`.

## Open Questions

None at this time.
