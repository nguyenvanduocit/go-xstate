# Ten games from the standard starting position

Clef won three games, Jev won one, and six were drawn. With one point for a
win and half a point for a draw, the score was **Clef 6–4 Jev**. Jev had lower
observed request latency and cost. This small batch does not establish a
general chess rating: Clef's three wins followed the same 15-ply sequence.

## Conditions

Run on 6 October 2026 using code commit
[`83c7639`](https://github.com/nguyenvanduocit/go-xstate/commit/83c76397962fa0ee7ac20f887aa3507ed6168e44).
All ten games entered `setup` with the standard board and no moves played.
Colors alternated, giving each model five games as White. Each game had a
1,000-ply limit, a 30-minute deadline, and a 45-second request timeout.

Each request offered all legal moves, without engine ranking or filtering.
The models received the same state format and instructions. Games ran
sequentially through OpenRouter's Decisions API, with no retries or fallback
moves. No temperature or seed was specified. The earlier batch using preset
opening positions is excluded. Exact settings are in [protocol.json](protocol.json).

## Observed results

| Metric | Clef | Jev 1.13 |
| --- | ---: | ---: |
| Wins / losses / draws | 3 / 1 / 6 | 1 / 3 / 6 |
| Score out of 10 | 6 | 4 |
| Accepted decisions | 421 | 418 |
| Median request latency | 708 ms | 399 ms |
| Mean request latency | 787.0 ms | 421.6 ms |
| Approximate mean cost per decision | $0.00019617 | $0.00003863 |
| Approximate accepted-response cost | $0.082586 | $0.016148 |

Jev's median latency was about 44% lower and its mean cost per decision about
80% lower. These measurements include this client and network, and the models
did not receive identical positions on every turn. They are observations of
these games, not controlled inference-speed or price benchmarks.

The sum of reported game costs was approximately **$0.098732** for 839 plies.
Per-move and per-game CLI output rounds costs to six decimal places, so the
model totals differ slightly from the game total. These costs cover accepted
responses in this batch, not prior experiments or account-wide charges.

| Game | White | Result | Plies | Ending |
| --- | --- | --- | ---: | --- |
| [01](01-standard-pair-1-clef-white.pgn) | Clef | Clef wins | 15 | Checkmate |
| [02](02-standard-pair-1-jev-white.pgn) | Jev | Draw | 164 | Insufficient material |
| [03](03-standard-pair-2-clef-white.pgn) | Clef | Clef wins | 15 | Checkmate |
| [04](04-standard-pair-2-jev-white.pgn) | Jev | Draw | 63 | Fivefold repetition |
| [05](05-standard-pair-3-clef-white.pgn) | Clef | Draw | 191 | Insufficient material |
| [06](06-standard-pair-3-jev-white.pgn) | Jev | Jev wins | 67 | Checkmate |
| [07](07-standard-pair-4-clef-white.pgn) | Clef | Draw | 137 | Fivefold repetition |
| [08](08-standard-pair-4-jev-white.pgn) | Jev | Draw | 62 | Fivefold repetition |
| [09](09-standard-pair-5-clef-white.pgn) | Clef | Clef wins | 15 | Checkmate |
| [10](10-standard-pair-5-jev-white.pgn) | Jev | Draw | 110 | Insufficient material |

Every game reached checkmate or an automatic draw. None hit the move limit,
timed out, or failed an API request. Eight move sequences were distinct;
games 01, 03, and 09 were identical. All four decisive games were won by White.
Repeating the same initial board limits the variety of positions tested.

## Check the records

[results.json](results.json) contains every recorded decision, latency, and
rounded cost. Each game has a PGN and CLI log. [summary.json](summary.json)
contains aggregate measurements and duplicate-game groups.

An independent rules implementation, `python-chess` 1.999 (engine package
`chess` 1.11.2), verified all 839 moves, SAN, PGN histories, final FENs, and
termination reasons. See [validation.json](validation.json). Repeat this
offline check from this directory with:

```sh
uv run --with python-chess==1.999 python verify.py
```

Clef scored higher here; Jev responded faster and cost less per decision.
A broader strength comparison needs more distinct games before treating
that score difference as reliable.
