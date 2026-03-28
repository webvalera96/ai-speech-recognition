package main

import (
	"github.com/webvalera96/ai-speech-recognition/internal/app"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		app.Module,
		fx.NopLogger,
	).Run()
}
