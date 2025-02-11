package log

import (
	"runtime"

	"github.com/rs/zerolog"
)

type fileAndLineHook struct{}

func (h fileAndLineHook) Run(e *zerolog.Event, level zerolog.Level, message string) {
	_, file, line, ok := runtime.Caller(3)
	if ok {
		e.Str("file", file).Int("line", line)
	}
}
