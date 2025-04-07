package misc

import "github.com/rs/zerolog/log"

type ZeroLogTemporalLoggerAdapter struct {
}

func (a ZeroLogTemporalLoggerAdapter) Info(m string) {
	log.Info().Str("subsystem", "temporal").Msg(m)
}

func (a ZeroLogTemporalLoggerAdapter) Debug(m string) {
	log.Debug().Str("subsystem", "temporal").Msg(m)
}

func (a ZeroLogTemporalLoggerAdapter) Error(m string, err error) {
	log.Error().Str("subsystem", "temporal").Err(err).Msg(m)
}
