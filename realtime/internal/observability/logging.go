package observability

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger

func InitLogger(level string) {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	Log = zerolog.New(os.Stdout).Level(lvl).With().
		Timestamp().
		Str("service", "realtime-engine").
		Str("version", "1.24.2").
		Logger()
}
