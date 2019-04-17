package cli

import (
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"

	"gopkg.in/src-d/go-log.v1"
)

// ProfilerOptions defines profiling flags. It is meant to be embedded in a
// command struct.
type ProfilerOptions struct {
	ProfilerHTTP          bool   `long:"profiler-http" env:"PROFILER_HTTP" description:"start HTTP profiler endpoint"`
	ProfilerBlockRate     int    `long:"profiler-block-rate" env:"PROFILER_BLOCK_RATE" default:"0" description:"runtime.SetBlockProfileRate parameter"`
	ProfilerMutexFraction int    `long:"profiler-mutex-rate" env:"PROFILER_MUTEX_FRACTION" default:"0" description:"runtime.SetMutexProfileFraction parameter"`
	ProfilerEndpoint      string `long:"profiler-endpoint" env:"PROFILER_ENDPOINT" description:"address to bind HTTP pprof endpoint to" default:"0.0.0.0:6061"`
	ProfilerCPU           string `long:"profiler-cpu" env:"PROFILER_CPU" description:"file where to write the whole execution CPU profile" default:""`
}

// Init initializes the profiler.
func (c ProfilerOptions) Init(a *App) error {
	runtime.SetBlockProfileRate(c.ProfilerBlockRate)
	runtime.SetMutexProfileFraction(c.ProfilerMutexFraction)

	if c.ProfilerHTTP {
		log.With(log.Fields{"address": c.ProfilerEndpoint}).
			Debugf("starting http pprof endpoint")
		registerPprof(a.DebugServeMux)
		lis, err := net.Listen("tcp", c.ProfilerEndpoint)
		if err != nil {
			return err
		}

		go func() {
			err := http.Serve(lis, a.DebugServeMux)
			if err != nil {
				log.Errorf(err, "failed to serve http pprof endpoint")
			}
		}()
	}

	if c.ProfilerCPU != "" {
		log.With(log.Fields{"file": c.ProfilerCPU}).Debugf("starting CPU pprof")

		cpu, err := os.Create(c.ProfilerCPU)
		if err != nil {
			return err
		}

		if err := pprof.StartCPUProfile(cpu); err != nil {
			return err
		}

		a.Defer(func() { pprof.StopCPUProfile() })
	}

	return nil
}
