package main

import (
	"os"
	"strings"

	"github.com/webvalera96/ai-speech-recognition/internal/app"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func main() {
	opts := []fx.Option{app.Module}
	if fxDebugEnabled() {
		opts = append(opts, fx.WithLogger(func() fxevent.Logger {
			return &fxevent.ConsoleLogger{W: os.Stdout}
		}))
	} else {
		opts = append(opts, fx.NopLogger)
	}
	fx.New(opts...).Run()
}

// fxDebugEnabled returns true when Fx startup logging should go to stdout.
// Set FX_LOG=debug (or true/1/yes), LOG_LEVEL=debug, or FX_DEBUG=1.
func fxDebugEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("FX_LOG"))) {
	case "debug", "true", "1", "yes":
		return true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("LOG_LEVEL")), "debug") {
		return true
	}
	return strings.TrimSpace(os.Getenv("FX_DEBUG")) == "1"
}
