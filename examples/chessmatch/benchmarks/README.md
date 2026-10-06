# Clef versus Jev: 20 games, including the original ten

Clef won **8**, Jev won **1**, and **11** games were drawn. The score is
**13.5–6.5** when a draw counts as half a point. This combines both batches;
the original ten games and their report remain unchanged.

All eight Clef wins repeated the same 15-ply game. Stockfish's per-move
analysis shows a smaller difference than the match score suggests, and the
gap becomes very small when each distinct game sequence is counted once.
The evidence supports an advantage for Clef in this repeated matchup, but
does not establish a general strength ranking.

## Both batches contribute to the result

| Games | Clef wins | Jev wins | Draws | Plies | Reported cost |
| --- | ---: | ---: | ---: | ---: | ---: |
| [Original 01–10](2026-10-06-standard/README.md) | 3 | 1 | 6 | 839 | $0.098732 |
| [Additional 11–20](2026-10-07-standard/README.md) | 5 | 0 | 5 | 646 | $0.081813 |
| **Combined 01–20** | **8** | **1** | **11** | **1,485** | **$0.180545** |

Every game started from the standard board through `setup`. Each model
played White ten times. Both batches used the same prompts, all legal moves,
a 1,000-ply cap, a 30-minute game deadline, and a 45-second request timeout.
No games hit a limit or failed an API request. Nine ended in checkmate,
seven in insufficient material, and four in fivefold repetition.

[combined/results.json](combined/results.json) contains all twenty original
game records with an added `batch` field. Nothing is filtered out of the
primary totals. Batch directories retain their PGNs, CLI logs, protocols,
and independently checked results. Costs are rounded accepted-response
charges; they exclude experiments outside these two batches.

## Stockfish scores the moves in both batches

Stockfish 19 analysed every position with 100,000 search nodes, one thread,
64 MiB hash, and MultiPV 1. Each new search resets the engine's hash and
search state. Evaluations are cached only when the full starting position
and move history match. Automatic terminal outcomes are scored exactly.
No engine suggestions were supplied to either model during play.

The primary quality measure is the decrease in Stockfish's expected game
score after a move, viewed from the player who moved. It uses the engine's
reported win/draw/loss distribution: `expected score = win + draw / 2`.
Negative decreases from search variation are clamped to zero.

| Measurement, all 20 games | Clef | Jev 1.13 |
| --- | ---: | ---: |
| Moves assessed | 744 | 741 |
| Mean expected-score loss per move, percentage points ↓ | 7.767 | 8.314 |
| Mean capped centipawn loss ↓ | 131.72 | 142.74 |
| Large errors: score loss of at least 20 percentage points | 104 | 102 |
| Large errors per 100 moves ↓ | 13.98 | 13.77 |
| Moves dropping an advantage from ≥90% to ≤60% expected score | 47 | 56 |
| Immediate checkmates missed / available | 0 / 8 | 0 / 1 |
| Median API request latency | 728 ms | 400 ms |
| Approximate mean cost per decision | $0.00020440 | $0.00003844 |

The error threshold is defined by this report, not an official Stockfish
annotation. Centipawn loss clips each position's evaluation to ±1,000 cp;
detected mates use the corresponding endpoint. This prevents mate sentinels
or extreme scores from dominating the average. Expected-score loss also
measures throwing away a win in a game that ultimately ends in a draw.

These expected scores describe Stockfish's evaluation model, **not the
probability that Clef or Jev will win**. Search has a finite budget, and
before/after evaluations can vary. Stockfish's documentation explains its
[evaluation scale and WDL calibration](https://official-stockfish.github.io/docs/stockfish-wiki/Stockfish-FAQ.html#interpretation-of-the-stockfish-evaluation).
Full history is retained; Stockfish can consider claimable threefold or
50-move draws, whereas these model players only stop at automatic draws.
Such positions are marked `claimable_draw` in the saved analysis.

## Repetition explains much of the match-score gap

There are **12 distinct complete move sequences** among the 20 games:

- Games 01, 03, 09, 11, 13, 15, 17, and 19 are the same Clef win.
- Games 12 and 16 are the same draw.
- Every other game has a distinct sequence.

As a supplementary view, counting each complete sequence once gives one
win per model and ten draws. Mean expected-score loss becomes **7.882**
percentage points for Clef and **7.933** for Jev; mean capped centipawn loss
becomes **133.64** versus **134.79**. This view does not replace the 20-game
result. Distinct games can still share many positions, so it is not a set
of independent samples either.

The models also encountered different positions and different numbers of
endgame moves. A small difference in their averages should not be treated
as an Elo estimate or a statistically established advantage. Latency includes
network and client conditions across the two runs.

## Why insufficient-material draws keep happening

In [game 12](2026-10-07-standard/12-standard-pair-6-jev-white.pgn), Clef
promoted a pawn to a queen, then played `61...Qe6+`. Jev replied `62.Kxe6`,
leaving only the two kings. Stockfish's expected score for Clef dropped from
1.0 to 0.5 on that queen move. Game 16 repeated this sequence.

The same failure appeared with `55...Qf4+ 56.Kxf4` in
[game 14](2026-10-07-standard/14-standard-pair-7-jev-white.pgn), and
`66...Qh8+ 67.Kxh8` in
[game 20](2026-10-07-standard/20-standard-pair-10-jev-white.pgn).
These four new insufficient-material draws followed Clef giving away its
promoted queen. They expose a conversion problem that the win/draw/loss
totals alone do not show.

Neither model missed an immediate checkmate in these 20 games: every legal
move was checked for mate-in-one, independently of Stockfish's search.
This does not rule out missed mating combinations requiring more moves.

## Reproduce or inspect the analysis

- [Combined summary](combined/summary.json): totals, per-batch results,
  duplicate groups, phase breakdowns, and both quality views.
- [Per-move analysis](combined/analysis.json): FEN, played move, engine best
  move, before/after scores, and error flags, plus engine settings and hash.
- [Combined validation](combined/validation.json): both batches checked,
  original records preserved, and every move represented in the analysis.

Phase labels are a reporting heuristic: endgame when remaining phase units
are at most eight (knight/bishop = 1, rook = 2, queen = 4, summed for both
sides); otherwise the first 20 plies are opening and later plies middlegame.

From this directory, with Stockfish 19 installed:

```sh
uv run --with python-chess==1.999 python analyze.py \
  2026-10-06-standard 2026-10-07-standard \
  --engine /path/to/stockfish --nodes 100000 \
  --cache /tmp/chess-stockfish-cache.json --output /tmp/chess-combined
```

Use a separate output directory to retain the published measurements.
The cache checks the engine binary hash and settings before reuse.
Engine builds can produce different finite-search evaluations. No model
API calls are made by this analysis command.

The five analyser tests cover both players' score perspectives, giving away
a queen, missed mate-in-one, automatic draws, and repetition-sensitive caching:

```sh
STOCKFISH=/path/to/stockfish uv run --with python-chess==1.999 \
  python -m unittest discover -s . -p test_analyze.py -v
```

Validate both batches and the combined records without Stockfish:

```sh
uv run --with python-chess==1.999 python verify_combined.py
```

Clef scores better across all recorded games; Jev uses less time and money
per decision. Outside the repeated short win, this dataset gives little
evidence of a substantial overall playing-strength difference.
