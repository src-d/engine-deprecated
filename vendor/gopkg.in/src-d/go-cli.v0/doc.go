/*
Package cli provides scaffolding for CLI applications. It is a convenience
wrapper for jessevdk/go-flags reducing boilerplate code.

The main entry point is the App type, created with the New function.

It provides:

  - Struct tags to specify command names and descriptions (see below).
  - Default version subcommand.
  - Flags and environment variables to setup logging with src-d/go-log.
  - Flags and environment variables to setup a http/pprof endpoint.
  - Signal handling.

Commands

Commands are defined with structs. For the general available struct tags, refer
to jessevdk/go-flags documentation at https://github.com/jessevdk/go-flags.

Additionally, every command struct must embed PlainCommand directly, or
indirectly through other struct (e.g. Command). The embedded field must be
defined with the struct tags name, short-description and long-description.
For example:

  type helloCommand struct {
    Command `name:"hello" short-description:"prints Hello World" long-description:"prints Hello World to standard output"`
  }

This will also work if nested:

  type myBaseCommand struct {
    Command
    Somethig string
  }

  type helloCommand struct {
    myBaseCommand `name:"hello" short-description:"prints Hello World" long-description:"prints Hello World to standard output"`
  }

Each defined command must be added to the application with the AddCommand
function.

Signal Handling

Signal handling is setup for any cancellable command. Cancellable commands
implement the ContextCommander interface instead of flags.Commander.

Examples

See full examples in the examples subpackage.

*/
package cli
