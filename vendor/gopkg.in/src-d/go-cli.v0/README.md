# go-cli [![GoDoc](https://godoc.org/gopkg.in/src-d/go-cli.v0?status.svg)](https://godoc.org/gopkg.in/src-d/go-cli.v0)  [![Build Status](https://travis-ci.org/src-d/go-cli.svg?branch=master)](https://travis-ci.org/src-d/go-cli) [![Build status](https://ci.appveyor.com/api/projects/status/xrcrytlq7ou5ll3r?svg=true)](https://ci.appveyor.com/project/mcuadros/go-cli) [![codecov](https://codecov.io/gh/src-d/go-cli/branch/master/graph/badge.svg)](https://codecov.io/gh/src-d/go-cli)

A thin wrapper around common libraries used in our CLI apps (`jessevdk/go-flags`, `src-d/go-log`, `pprof`) to reduce boilerplate code and help in being more homogeneous with respect how our CLI work and look like.

It provides:
- Struct tags to specify command names and descriptions (see below).
- Default version subcommand.
- Flags and environment variables to setup logging with src-d/go-log.
- Flags and environment variables to setup a http/pprof endpoint.
- Signal handling.

For further details, look at `doc.go`.

## License

Apache License Version 2.0, see [LICENSE](LICENSE).
