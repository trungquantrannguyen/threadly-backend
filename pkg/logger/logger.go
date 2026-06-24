package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func New(serviceName string, appEnv string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	if appEnv == "production" {
		return zerolog.New(os.Stdout).With().Timestamp().Str("service", serviceName).Logger()
	}

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "15:04:05",
	}

	return zerolog.New(consoleWriter).With().Timestamp().Str("service", serviceName).Logger()
}
