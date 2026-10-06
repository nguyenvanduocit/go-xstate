package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/nguyenvanduocit/go-xstate/examples/badukmatch"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("baduk", flag.ContinueOnError)
	white := flags.String("white", badukmatch.Jev, "White's OpenRouter decision model")
	black := flags.String("black", badukmatch.Clef, "Black's OpenRouter decision model")
	moves := flags.Int("max-moves", 1000, "maximum moves, including passes (must be positive)")
	size := flags.Int("size", 9, "board size (2 through 19)")
	komi := flags.Float64("komi", 7.5, "points added to White's area score")
	demo := flags.Bool("demo", false, "use deterministic local choices without API calls")
	timeout := flags.Duration("request-timeout", 45*time.Second, "timeout for each model request")
	deadline := flags.Duration("timeout", 30*time.Minute, "timeout for the entire match")
	sgf := flags.String("sgf", "", "optional output SGF file (must not already exist)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *moves <= 0 || *deadline <= 0 || *timeout <= 0 {
		return fmt.Errorf("max-moves and timeouts must be positive")
	}
	if *size < 2 || *size > 19 {
		return fmt.Errorf("board size must be between 2 and 19")
	}
	choose := badukmatch.ChooseMove(badukmatch.DemoChoose)
	if *demo {
		*black, *white = "scripted-demo-black", "scripted-demo-white"
	}
	if !*demo {
		key := os.Getenv("OPENROUTER_KEY")
		if key == "" {
			key = os.Getenv("OPENROUTER_API_KEY")
		}
		client, err := badukmatch.NewOpenRouter(key, *timeout)
		if err != nil {
			return err
		}
		choose = client.Choose
	}
	cfg := badukmatch.Config{Size: *size, Komi: *komi, WhiteModel: *white, BlackModel: *black, MaxMoves: *moves}
	if _, err := badukmatch.NewMachine(cfg, choose); err != nil {
		return err
	}
	var err error
	var file *os.File
	keepSGF := false
	if *sgf != "" {
		file, err = os.OpenFile(*sgf, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return fmt.Errorf("save SGF: %w", err)
		}
		defer func() {
			_ = file.Close()
			if !keepSGF {
				_ = os.Remove(file.Name())
			}
		}()
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *deadline)
	defer cancel()
	if *demo {
		fmt.Println("DEMO: local scripted choices; no model requests")
	}
	fmt.Printf("Black: %s\nWhite: %s\nBoard: %dx%d, komi %g\nLimit: %d moves\n", *black, *white, *size, *size, *komi, *moves)
	match, playErr := badukmatch.Play(ctx, cfg, choose, func(move badukmatch.Move) {
		fmt.Printf("%3d %-5s %-22s %-5s confidence=%.3f cost=$%.6f latency=%dms\n",
			move.Number, move.Side, move.Model, move.Point, move.Confidence, move.CostUSD, move.LatencyMS)
	})
	fmt.Printf("\n%s\nResult: %s (%s), %d moves, reported cost $%.6f\n\n%s\n",
		match.Board, match.Result, match.Reason, len(match.Moves), match.CostUSD, match.SGF)
	if file != nil && match.SGF != "" {
		keepSGF = true
		_, writeErr := fmt.Fprintln(file, match.SGF)
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
