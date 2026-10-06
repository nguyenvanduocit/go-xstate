"""Run with STOCKFISH pointing to an installed engine; no model API calls."""

import os
import unittest

import chess
import chess.engine

from analyze import Evaluator, mate_in_one_moves


class EvaluatorTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.engine = chess.engine.SimpleEngine.popen_uci(os.environ["STOCKFISH"])
        cls.engine.configure({"Threads": 1, "Hash": 64, "UCI_ShowWDL": True})

    @classmethod
    def tearDownClass(cls):
        cls.engine.quit()

    def setUp(self):
        self.evaluator = Evaluator(self.engine, 20000, {})

    def test_checkmate_scores_from_the_moving_players_perspective(self):
        for moves, finish, winner in [
            (["f2f3", "e7e5", "g2g4"], "d8h4", chess.BLACK),
            (["e2e4", "f7f6", "d2d4", "g7g5"], "d1h5", chess.WHITE),
        ]:
            with self.subTest(winner=winner):
                board = chess.Board()
                for move in moves:
                    board.push_uci(move)
                result = self.evaluator.move(board, {"uci": finish, "model": "test"})
                self.assertTrue(board.is_checkmate())
                self.assertEqual(result["expected_score_before"], 1)
                self.assertEqual(result["expected_score_after"], 1)
                self.assertEqual(result["expected_score_loss"], 0)
                self.assertEqual(result["capped_cp_loss"], 0)
                self.assertFalse(result["missed_mate_in_one"])
                self.assertEqual(result["after"]["expected_white"], float(winner))

    def test_missed_mate_in_one_is_verified_by_rules(self):
        board = chess.Board()
        for move in ["f2f3", "e7e5", "g2g4"]:
            board.push_uci(move)
        before = board.fen()
        self.assertEqual(mate_in_one_moves(board), ["d8h4"])
        self.assertEqual(board.fen(), before)
        result = self.evaluator.move(board, {"uci": "a7a6", "model": "test"})
        self.assertTrue(result["missed_mate_in_one"])
        self.assertFalse(board.is_checkmate())

    def test_last_pawn_capture_is_exact_draw(self):
        board = chess.Board("8/7P/3K2k1/8/8/8/8/8 b - - 0 82")
        result = self.evaluator.move(board, {"uci": "g6h7", "model": "test"})
        self.assertTrue(board.is_insufficient_material())
        self.assertEqual(result["after"]["expected_white"], 0.5)
        self.assertEqual(result["after"]["capped_cp_white"], 0)
        self.assertEqual(result["after"]["nodes"], 0)

    def test_hanging_the_only_queen_loses_winning_advantage_for_either_color(self):
        white = chess.Board("7k/8/8/8/4Q3/8/8/K7 w - - 0 1")
        for board, move in [(white, "e4h7"), (white.mirror(), "e5h2")]:
            with self.subTest(side=board.turn):
                self.assertTrue(board.is_valid())
                result = self.evaluator.move(board, {"uci": move, "model": "test"})
                self.assertEqual(result["expected_score_before"], 1)
                self.assertEqual(result["expected_score_after"], 0.5)
                self.assertEqual(result["expected_score_loss"], 0.5)
                self.assertTrue(result["large_error"])
                self.assertTrue(result["lost_advantage"])
                self.assertGreater(result["capped_cp_loss"], 0)

    def test_repetition_history_is_part_of_the_cache_key(self):
        board = chess.Board()
        for move in ["g1f3", "g8f6", "f3g1", "f6g8"] * 2:
            board.push_uci(move)
        fresh = chess.Board(board.fen())
        self.assertTrue(board.can_claim_threefold_repetition())
        self.assertFalse(fresh.can_claim_threefold_repetition())
        self.assertTrue(self.evaluator.evaluate(board)["claimable_draw"])
        self.assertFalse(self.evaluator.evaluate(fresh)["claimable_draw"])
        self.assertEqual(len(self.evaluator.cache), 2)
        self.evaluator.evaluate(board)
        self.assertEqual(len(self.evaluator.cache), 2)


if __name__ == "__main__":
    unittest.main()
