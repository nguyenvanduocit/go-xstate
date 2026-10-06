# Additional games 11–20

This batch adds ten games to the original ten. Clef won five and five were
drawn, over 646 plies. Reported accepted-response cost was approximately
$0.081813. All five wins repeat the 15-ply game seen in the first batch.

The [combined report](../README.md) includes all twenty games and Stockfish
analysis of both batches. The original batch remains unchanged.

Run on 7 October 2026, with the same standard starting board, model prompts,
and chess implementation as the original batch. Colors alternate, each
model plays White five times, and the limits remain 1,000 plies, 30 minutes
per game, and 45 seconds per request. See [protocol.json](protocol.json).

| Game | White | Result | Plies | Ending |
| --- | --- | --- | ---: | --- |
| [11](11-standard-pair-6-clef-white.pgn) | Clef | Clef wins | 15 | Checkmate |
| [12](12-standard-pair-6-jev-white.pgn) | Jev | Draw | 123 | Insufficient material |
| [13](13-standard-pair-7-clef-white.pgn) | Clef | Clef wins | 15 | Checkmate |
| [14](14-standard-pair-7-jev-white.pgn) | Jev | Draw | 111 | Insufficient material |
| [15](15-standard-pair-8-clef-white.pgn) | Clef | Clef wins | 15 | Checkmate |
| [16](16-standard-pair-8-jev-white.pgn) | Jev | Draw | 123 | Insufficient material |
| [17](17-standard-pair-9-clef-white.pgn) | Clef | Clef wins | 15 | Checkmate |
| [18](18-standard-pair-9-jev-white.pgn) | Jev | Draw | 81 | Fivefold repetition |
| [19](19-standard-pair-10-clef-white.pgn) | Clef | Clef wins | 15 | Checkmate |
| [20](20-standard-pair-10-jev-white.pgn) | Jev | Draw | 133 | Insufficient material |

No game hit a limit or failed a request. [results.json](results.json) retains
all decisions and rounded request measurements. Each game also has its PGN
and CLI log. [validation.json](validation.json) records the independent
`python-chess` check of every move and terminal outcome. Repeat that check:

```sh
uv run --with python-chess==1.999 python verify.py
```
