"""Validate both batches and ensure the combined data preserves every record."""

import json
import pathlib
import subprocess
import sys

import chess


root = pathlib.Path(__file__).resolve().parent
batches = ["2026-10-06-standard", "2026-10-07-standard"]
source_records = []
validations = []
for batch in batches:
    output = subprocess.check_output([sys.executable, str(root / batch / "verify.py")], text=True)
    validations.append(dict(json.loads(output), batch=batch))
    for record in json.loads((root / batch / "results.json").read_text()):
        source_records.append(dict(record, batch=batch))

combined = json.loads((root / "combined/results.json").read_text())
analysis = json.loads((root / "combined/analysis.json").read_text())
summary = json.loads((root / "combined/summary.json").read_text())
assert combined == source_records, "combined records changed, dropped, or reordered source data"
assert len(combined) == 20
assert len({g["game_id"] for g in combined}) == 20
assert sum(g["white"] == "cloudflare/clef" for g in combined) == 10
assert sum(g["white"] == "typesafe/jev-1.13" for g in combined) == 10
assert len(analysis["games"]) == len(combined)
for game, scored in zip(combined, analysis["games"]):
    assert game["game_id"] == scored["game_id"]
    assert game["batch"] == scored["batch"]
    assert len(game["moves"]) == len(scored["moves"])
    board = chess.Board(game["initial_fen"])
    for played, assessed in zip(game["moves"], scored["moves"]):
        for field in ["ply", "model", "side", "san", "uci"]:
            assert played[field] == assessed[field], (game["game_id"], field)
        expected_model = game["white"] if board.turn else game["black"]
        assert assessed["model"] == expected_model
        assert assessed["fen_before"] == board.fen()
        assert chess.Move.from_uci(assessed["before"]["best_move"]) in board.legal_moves
        board.push_uci(played["uci"])
        assert 0 <= assessed["expected_score_loss"] <= 1
        assert 0 <= assessed["capped_cp_loss"] <= 2000
    assert board.outcome(claim_draw=False).result() == game["result"]

assert summary["games"] == len(combined)
assert summary["total_plies"] == sum(g["plies"] for g in combined)
assert summary["total_reported_cost_usd"] == round(sum(g["cost_usd"] for g in combined), 6)
for model, stats in summary["models"].items():
    assert stats["wins"] == sum(g["winner"] == model for g in combined)
    assert stats["draws"] == sum(g["result"] == "1/2-1/2" for g in combined)
    assert stats["decisions"] == sum(m["model"] == model for g in combined for m in g["moves"])
    assert stats["quality"]["moves"] == stats["decisions"]
    assert sum(p["moves"] for p in stats["quality_by_phase"].values()) == stats["decisions"]

print(json.dumps({
    "completed_games_checked": len(combined),
    "total_plies_checked": summary["total_plies"],
    "all_source_records_preserved": True,
    "all_moves_represented_in_analysis": True,
    "white_games_per_model": 10,
    "batches": validations,
}, indent=2))
