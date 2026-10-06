"""Combine saved batches and score their moves offline with Stockfish."""

import argparse
import collections
import hashlib
import json
import pathlib
import statistics

import chess
import chess.engine


MODELS = ("cloudflare/clef", "typesafe/jev-1.13")


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2) + "\n")


def phase(board):
    units = sum(len(board.pieces(piece, side)) * weight
                for side in chess.COLORS
                for piece, weight in [(chess.KNIGHT, 1), (chess.BISHOP, 1),
                                      (chess.ROOK, 2), (chess.QUEEN, 4)])
    if units <= 8:
        return "endgame"
    return "opening" if board.ply() < 20 else "middlegame"


def mate_in_one_moves(board):
    mates = []
    for move in list(board.legal_moves):
        board.push(move)
        if board.is_checkmate():
            mates.append(move.uci())
        board.pop()
    return mates


class Evaluator:
    def __init__(self, engine, nodes, cache):
        self.engine = engine
        self.nodes = nodes
        self.cache = cache

    def evaluate(self, board):
        # Repetition and the halfmove clock matter; FEN alone loses move history.
        history = board.root().fen() + " " + " ".join(m.uci() for m in board.move_stack)
        key = hashlib.sha256(history.encode()).hexdigest()
        if key in self.cache:
            return self.cache[key]
        outcome = board.outcome(claim_draw=False)
        if outcome is not None:
            expected = 0.5 if outcome.winner is None else float(outcome.winner)
            evaluation = {"cp_white": 0 if outcome.winner is None else None,
                          "mate_white": None if outcome.winner is None else 0,
                          "capped_cp_white": int((expected - 0.5) * 2000),
                          "expected_white": expected, "best_move": None,
                          "depth": 0, "nodes": 0,
                          "terminal": outcome.termination.name,
                          "claimable_draw": False}
        else:
            # A new game resets the hash and search heuristics for each position.
            info = self.engine.analyse(board, chess.engine.Limit(nodes=self.nodes),
                                       game=object())
            score = info["score"].white()
            evaluation = {"cp_white": score.score(), "mate_white": score.mate(),
                          "capped_cp_white": max(-1000, min(1000, score.score(mate_score=10000))),
                          "expected_white": info["wdl"].white().expectation(),
                          "best_move": info["pv"][0].uci(),
                          "depth": info["depth"], "nodes": info["nodes"],
                          "terminal": None, "claimable_draw": board.can_claim_draw()}
        self.cache[key] = evaluation
        return evaluation

    def move(self, board, record):
        move = chess.Move.from_uci(record["uci"])
        if move not in board.legal_moves:
            raise ValueError(f"illegal move {move} in {board.fen()}")
        turn = board.turn
        before = self.evaluate(board)
        mates = mate_in_one_moves(board)
        result = {"ply": board.ply() + 1, "model": record["model"],
                  "side": "white" if turn else "black", "uci": move.uci(),
                  "san": board.san(move), "phase": phase(board),
                  "fen_before": board.fen(), "mate_in_one_choices": mates,
                  "missed_mate_in_one": bool(mates) and move.uci() not in mates}
        board.push(move)
        after = self.evaluate(board)
        expected_before = before["expected_white"] if turn else 1 - before["expected_white"]
        expected_after = after["expected_white"] if turn else 1 - after["expected_white"]
        sign = 1 if turn else -1
        loss = expected_before - expected_after
        result.update(before=before, after=after,
                      expected_score_before=round(expected_before, 6),
                      expected_score_after=round(expected_after, 6),
                      expected_score_loss=round(max(0, loss), 6),
                      capped_cp_loss=max(0, sign * (before["capped_cp_white"] - after["capped_cp_white"])),
                      large_error=loss >= 0.2 - 1e-9,
                      lost_advantage=expected_before >= 0.9 and expected_after <= 0.6)
        return result


def quality(moves):
    if not moves:
        return {"moves": 0}
    large_errors = sum(m["large_error"] for m in moves)
    return {"moves": len(moves),
            "mean_expected_score_loss_pp": round(100 * statistics.mean(m["expected_score_loss"] for m in moves), 3),
            "mean_capped_cp_loss": round(statistics.mean(m["capped_cp_loss"] for m in moves), 2),
            "large_errors": large_errors,
            "large_errors_per_100_moves": round(100 * large_errors / len(moves), 2),
            "mate_in_one_opportunities": sum(bool(m["mate_in_one_choices"]) for m in moves),
            "missed_mate_in_one": sum(m["missed_mate_in_one"] for m in moves),
            "lost_advantage_moves": sum(m["lost_advantage"] for m in moves)}


def summarize(games, analyses):
    sequences = collections.defaultdict(list)
    for game in games:
        sequences[(game["white"], tuple(m["uci"] for m in game["moves"]))].append(game["game_id"])
    representatives = {ids[0] for ids in sequences.values()}
    models = {}
    for model in MODELS:
        moves = [m for g in games for m in g["moves"] if m["model"] == model]
        scored = [m for g in analyses for m in g["moves"] if m["model"] == model]
        unique_scored = [m for g in analyses if g["game_id"] in representatives
                         for m in g["moves"] if m["model"] == model]
        wins = sum(g["winner"] == model for g in games)
        losses = sum(g["winner"] is not None and g["winner"] != model for g in games)
        draws = sum(g["result"] == "1/2-1/2" for g in games)
        cost = sum(m["cost_usd_rounded"] for m in moves)
        distinct_games = [g for g in games if g["game_id"] in representatives]
        distinct_results = {
            "wins": sum(g["winner"] == model for g in distinct_games),
            "losses": sum(g["winner"] is not None and g["winner"] != model for g in distinct_games),
            "draws": sum(g["result"] == "1/2-1/2" for g in distinct_games),
        }
        models[model] = {"wins": wins, "losses": losses, "draws": draws,
                         "score": wins + draws / 2,
                         "unfinished": len(games) - wins - losses - draws,
                         "decisions": len(moves),
                         "median_latency_ms": statistics.median(m["latency_ms"] for m in moves),
                         "mean_latency_ms": round(statistics.mean(m["latency_ms"] for m in moves), 1),
                         "accepted_cost_usd_rounded": round(cost, 6),
                         "mean_cost_usd_rounded": round(cost / len(moves), 8),
                         "quality": quality(scored),
                         "distinct_game_results": distinct_results,
                         "quality_distinct_games": quality(unique_scored),
                         "quality_by_phase": {p: quality([m for m in scored if m["phase"] == p])
                                              for p in ["opening", "middlegame", "endgame"]}}
    return {"games": len(games), "total_plies": sum(g["plies"] for g in games),
            "total_reported_cost_usd": round(sum(g["cost_usd"] for g in games), 6),
            "unique_move_sequences": len(sequences),
            "duplicate_groups": [ids for ids in sequences.values() if len(ids) > 1],
            "terminations": dict(collections.Counter(g["reason"] for g in games)),
            "models": models}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("batches", nargs="+", type=pathlib.Path)
    parser.add_argument("--engine", type=pathlib.Path, required=True)
    parser.add_argument("--nodes", type=int, default=100000)
    parser.add_argument("--cache", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    args = parser.parse_args()
    if args.nodes <= 0:
        parser.error("--nodes must be positive")
    games = []
    for batch in args.batches:
        for record in json.loads((batch / "results.json").read_text()):
            games.append(dict(record, batch=batch.name))
    if len({g["game_id"] for g in games}) != len(games):
        raise ValueError("duplicate game IDs across batches")
    args.output.mkdir(parents=True, exist_ok=True)
    with chess.engine.SimpleEngine.popen_uci(str(args.engine)) as engine:
        engine.configure({"Threads": 1, "Hash": 64, "UCI_ShowWDL": True})
        config = {"engine": engine.id["name"],
                  "engine_sha256": hashlib.sha256(args.engine.read_bytes()).hexdigest(),
                  "python_chess_version": chess.__version__,
                  "nodes_per_position": args.nodes, "threads": 1, "hash_mb": 64,
                  "multipv": 1, "reset_between_positions": True,
                  "history": "full move history; automatic terminal outcomes scored exactly",
                  "tablebases": False}
        cache = {"config": config, "positions": {}}
        if args.cache.exists():
            cache = json.loads(args.cache.read_text())
            if cache["config"] != config:
                raise ValueError("cache engine/config mismatch; use a new cache file")
        evaluator = Evaluator(engine, args.nodes, cache["positions"])
        analyses = []
        for game in games:
            board = chess.Board(game["initial_fen"])
            moves = []
            for record in game["moves"]:
                moves.append(evaluator.move(board, record))
                if len(moves) % 50 == 0:
                    print(game["game_id"], "analysed", len(moves), flush=True)
            analyses.append({"game_id": game["game_id"], "batch": game["batch"], "moves": moves})
            write_json(args.cache, cache)
            print("DONE", game["game_id"], len(moves), flush=True)
    summary = summarize(games, analyses)
    summary["batches"] = {batch.name: summarize([g for g in games if g["batch"] == batch.name],
                                               [g for g in analyses if g["batch"] == batch.name])
                          for batch in args.batches}
    write_json(args.output / "results.json", games)
    write_json(args.output / "analysis.json", {"config": config, "games": analyses})
    write_json(args.output / "summary.json", summary)
    print("WROTE", args.output, flush=True)


if __name__ == "__main__":
    main()
