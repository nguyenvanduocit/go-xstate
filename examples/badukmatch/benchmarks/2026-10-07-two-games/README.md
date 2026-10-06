# Two Go games: one win each after early passes

On 7 October 2026, Clef and Jev played two games from an empty 9×9 board,
swapping colors for game two. Both games used core Tromp–Taylor rules,
7.5 komi, every legal placement plus pass, and a 1,000-move cap.
They ran concurrently with a 30-minute match deadline and 45-second request
deadline. There were no retries or substituted moves.

| Game | Black | White | Winner | Moves including passes | Reported cost |
| --- | --- | --- | --- | ---: | ---: |
| [1](01.sgf) | Clef | Jev | Jev, W+6.5 | 15 | $0.004789 |
| [2](02.sgf) | Jev | Clef | Clef, W+6.5 | 17 | $0.004914 |

Total: **Clef 1 win, Jev 1 win; 32 moves; approximately $0.009703**.
Costs come from the API's reported values and are rounded to six decimals.
Summing individually rounded move costs can differ from the match total.

Both games ended with White passing and Black immediately passing too.
Neither game had a capture or self-capture. Game one finished with seven
black stones and six white stones; game two had eight black and seven white.
All empty points were neutral under area scoring. Black's one-point board
lead became a 6.5-point loss after White's komi.

These two short games do not establish equal playing strength. Both models
accepted an early end, and Black passed while behind after komi in each game.
The move cap and request deadlines did not cause either ending.

The [logs for game one](01.log) and [game two](02.log) contain every accepted
move, confidence, reported cost, latency, and final board.
[results.json](results.json) summarizes both games;
[protocol.json](protocol.json) records the configuration.
The earlier two-move API check remains separate and is not counted here.

An independent replay using sgfmill 1.1.1 checked all 32 moves, captures,
positional superko, log/SGF agreement, final boards, and scoring.
[validation.json](validation.json) records the checks. To repeat them:

```sh
uv run --with sgfmill==1.1.1 python verify.py
```
