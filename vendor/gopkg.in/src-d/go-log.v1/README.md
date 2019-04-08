# go-log [![GoDoc](https://godoc.org/gopkg.in/src-d/go-log.v1?status.svg)](https://godoc.org/github.com/src-d/go-log) [![Build Status](https://travis-ci.org/src-d/go-log.svg)](https://travis-ci.org/src-d/go-log) [![Build status](https://ci.appveyor.com/api/projects/status/15cdr1nk890qpk7g?svg=true)](https://ci.appveyor.com/project/mcuadros/go-log) [![codecov.io](https://codecov.io/github/src-d/go-log/coverage.svg)](https://codecov.io/github/src-d/go-log) [![Go Report Card](https://goreportcard.com/badge/github.com/src-d/go-log)](https://goreportcard.com/report/github.com/src-d/go-log)

Log is a generic logging library based on logrus (this may change in the
future), that minimize the exposure of src-d projects to logrus or any other
logging library, as well defines the standard for configuration and usage of the
logging libraries in the organization.

Installation
------------

The recommended way to install *go-log* is:

```
go get -u gopkg.in/src-d/go-log.v1/...
```

Configuration
-------------

The configuration should be done always using environment variables. The list
of available variables is:

- `LOG_LEVEL`: Reporting level, values are "info", "debug", "warning" or "error".
- `LOG_FORMAT`: Format of the log lines, values are "text" or "json", by default "text" is used. unless a terminal can't be detected, in this case, "json" is used instead.
- `LOG_FIELDS`: Fields in JSON format to be included in all the loggers.
- `LOG_FORCE_FORMAT`: If true the fact of being in a terminal or not is ignored.

> By default the logging is disabled if go-log is being executed in tests.

Usage
-----

### Basic usage

The most basic form of logging is made using the `Infof`, `Debugf`, `Warningf`
and `Errorf` functions at the top level of the packages.

```go
log.Infof("The answer to life, the universe and everything is %d", 42)
// INFO The answer to life, the universe and everything is 42
```

These functions use the `DefaultLogger` a logger lazy instanced when this method
are called. This logger reads the configuration from the environment variables.

### Logger instantiation

If you prefer to keep a reference to the `Logger`, in your packages or structs
to have more control over it (for example for tests). A default `Logger`, can
be instantiated using the `New` method.

```go
logger := log.New(nil)
logger.Infof("The answer to life, the universe and everything is %d", 42)
// INFO The answer to life, the universe and everything is 42
```

Also, a new `Logger` can be created from other `Logger` in order to have
contextual information, using the method `Logger.New`

```go
logger := log.New(nil)

authorLogger := logger.New(log.Field{"author": "Douglas Adams"})
bookLogger.Infof("The Hitchhiker's Guide to the Galaxy")
bookLogger.Infof("Life, the Universe and Everything")
// INFO The Hitchhiker's Guide to the Galaxy author=Douglas Adams
// INFO Life, the Universe and Everything author=Douglas Adams
```

Or if you just want to add contextual information `Logger.New` to one log line
you can use the `Logger.With` method.

```go
logger := log.New(nil)

authorLogger := logger.New(log.Field{"author": "Douglas Adams"})
bookLogger.With(log.Fields{"isbn": "0-330-25864-8"}).Infof("The Hitchhiker's Guide to the Galaxy")
bookLogger.With(log.Fields{"isbn": "0-345-39182-9"}).Infof("Life, the Universe and Everything")
// INFO The Hitchhiker's Guide to the Galaxy author=Douglas Adams isbn=0-330-25864-8
// INFO Life, the Universe and Everything author=Douglas Adams isbn=0-345-39182-9
```

### Logging errors

In `go-log` the errors are logged using the function `Logger.Errorf`:

```go
logger, _ := log.New()

_, err := http.Get("https://en.wikipedia.org/wiki/Douglas_Adams")
if err != nil {
    logger.Errorf(err, "unable to retrieve page")
}
```

License
-------
Apache License Version 2.0, see [LICENSE](LICENSE)
