package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/nguyenvanduocit/go-xstate/examples/chessmatch"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("chess", flag.ContinueOnError)
	white := flags.String("white", chessmatch.Clef, "White's OpenRouter decision model")
	black := flags.String("black", chessmatch.Jev, "Black's OpenRouter decision model")
	plies := flags.Int("max-plies", 1000, "maximum half-moves (must be positive)")
	fen := flags.String("fen", "", "starting FEN; empty uses the standard position")
	timeout := flags.Duration("request-timeout", 45*time.Second, "timeout for each model request")
	deadline := flags.Duration("timeout", 30*time.Minute, "timeout for the entire match")
	pgn := flags.String("pgn", "", "optional output PGN file (must not already exist)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *plies <= 0 || *deadline <= 0 {
		return fmt.Errorf("max-plies and timeout must be positive")
	}
	key := os.Getenv("OPENROUTER_KEY")
	if key == "" {
		key = os.Getenv("OPENROUTER_API_KEY")
	}
	client, err := chessmatch.NewOpenRouter(key, *timeout)
	if err != nil {
		return err
	}
	var file *os.File
	keepPGN := false
	if *pgn != "" {
		file, err = os.OpenFile(*pgn, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return fmt.Errorf("save PGN: %w", err)
		}
		defer func() {
			_ = file.Close()
			if !keepPGN {
				_ = os.Remove(file.Name())
			}
		}()
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *deadline)
	defer cancel()
	fmt.Printf("White: %s\nBlack: %s\nLimit: %d plies\n", *white, *black, *plies)
	match, playErr := chessmatch.Play(ctx, chessmatch.Config{
		WhiteModel: *white, BlackModel: *black, MaxPlies: *plies, FEN: *fen,
	}, client.Choose, func(move chessmatch.Ply) {
		fmt.Printf("%3d %-5s %-22s %-7s %s confidence=%.3f cost=$%.6f latency=%dms\n",
			move.Number, move.Side, move.Model, move.SAN, move.UCI, move.Confidence, move.CostUSD, move.LatencyMS)
	})
	fmt.Printf("\n%s\nResult: %s (%s), %d plies, reported cost $%.6f\nFEN: %s\n\n%s\n",
		match.Board, match.Result, match.Reason, len(match.Moves), match.CostUSD, match.FEN, match.PGN)
	if file != nil && match.PGN != "" {
		keepPGN = true
		_, writeErr := fmt.Fprintln(file, match.PGN)
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return playErr
}
