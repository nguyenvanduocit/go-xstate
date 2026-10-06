"""Generate independent capture/area fixtures using sgfmill 1.1.1.

sgfmill implements self-capture but not superko; apply positional history here.
Run: uv run --with sgfmill==1.1.1 python testdata/generate.py
"""

import json
import pathlib
import random

from sgfmill import boards


def cells(board):
    return "".join({None: ".", "b": "X", "w": "O"}[board.get(row, col)]
                   for row in range(board.side - 1, -1, -1)
                   for col in range(board.side))


def generate(size, length, seed):
    rng = random.Random(seed)
    board = boards.Board(size)
    history = {cells(board)}
    steps = []
    previous_pass = False
    captures = self_captures = 0
    for ply in range(length):
        color = "b" if ply % 2 == 0 else "w"
        choices = []
        positions = {}
        for row in range(size - 1, -1, -1):
            for col in range(size):
                if board.get(row, col) is not None:
                    continue
                candidate = board.copy()
                candidate.play(row, col, color)
                if cells(candidate) not in history:
                    move = "ABCDEFGHJKLMNOPQRST"[col] + str(row + 1)
                    choices.append(move)
                    positions[move] = candidate
        choices.append("pass")
        if ply >= length - 2 or not positions or (not previous_pass and rng.random() < 0.04):
            move = "pass"
        else:
            move = rng.choice(choices[:-1])
        before = cells(board)
        if move != "pass":
            board = positions[move]
            history.add(cells(board))
            own, opponent = ("X", "O") if color == "b" else ("O", "X")
            captures += before.count(opponent) - cells(board).count(opponent)
            self_captures += before.count(own) + 1 - cells(board).count(own)
        steps.append({"before": before, "side": "black" if color == "b" else "white",
                      "choices": choices, "move": move, "after": cells(board),
                      "area_difference": board.area_score()})
        if previous_pass and move == "pass":
            break
        previous_pass = move == "pass"
    return {"size": size, "seed": seed, "captured_stones": captures,
            "self_captured_stones": self_captures, "steps": steps}


fixture = {"source": "sgfmill 1.1.1 (capture and area scoring), positional history filter",
           "cases": [generate(3, 60, 13), generate(5, 120, 29), generate(9, 200, 47)]}
path = pathlib.Path(__file__).with_name("rules.json")
path.write_text(json.dumps(fixture, indent=2) + "\n")
print(json.dumps([{k: v for k, v in case.items() if k != "steps"} | {"moves": len(case["steps"])}
                  for case in fixture["cases"]]))
