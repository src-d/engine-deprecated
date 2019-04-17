package cli

import (
	"fmt"
	"net/http"
	"os"
	"reflect"

	"github.com/jessevdk/go-flags"
)

type DeferFunc func()

// App defines the CLI application that will be run.
type App struct {
	Parser *flags.Parser

	// DebugServeMux is serves debug endpoints. It used to attach the http/pprof
	// endpoint if enabled, and can be used to handle other debug endpoints.
	DebugServeMux *http.ServeMux

	// deferFuncs holds the functions to be called when the command finishes.
	deferFuncs []DeferFunc
}

// New creates a new App, including default values and sub commands.
func New(name, version, build, description string) *App {
	app := NewNoDefaults(name, description)

	app.AddCommand(&VersionCommand{
		Name:    name,
		Version: version,
		Build:   build,
	})

	return app
}

// NewNoDefaults creates a new App, without any of the default sub commands.
func NewNoDefaults(name, description string) *App {
	parser := flags.NewNamedParser(name, flags.Default)
	parser.LongDescription = description
	app := &App{
		Parser:        parser,
		DebugServeMux: http.NewServeMux(),
	}

	app.Parser.CommandHandler = app.commandHandler

	return app
}

// Run runs the app with the given command line arguments. In order to reduce
// boilerplate, RunMain should be used instead.
func (a *App) Run(args []string) error {
	defer a.callDefer()

	if _, err := a.Parser.ParseArgs(args[1:]); err != nil {
		if err, ok := err.(*flags.Error); ok {
			if err.Type == flags.ErrHelp {
				return nil
			}

			a.Parser.WriteHelp(os.Stderr)
		}

		return err
	}

	return nil
}

// RunMain runs the application with os.Args and if there is any error, it
// exits with error code 1.
func (a *App) RunMain() {
	if err := a.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

// Defer adds a function to be called after the command is executed. The
// functions added are called in reverse order.
func (a *App) Defer(d DeferFunc) {
	a.deferFuncs = append(a.deferFuncs, d)
}

func (a *App) callDefer() {
	for i := len(a.deferFuncs) - 1; i >= 0; i-- {
		f := a.deferFuncs[i]
		if f != nil {
			f()
		}
	}
}

func (a *App) commandHandler(cmd flags.Commander, args []string) error {
	if v, ok := cmd.(Initializer); ok {
		if err := v.Init(a); err != nil {
			return err
		}
	}

	if v, ok := cmd.(ContextCommander); ok {
		return executeContextCommander(v, args)
	}

	return cmd.Execute(args)
}

func getStructType(data interface{}) (reflect.Type, error) {
	typ := reflect.TypeOf(data)
	if typ == nil {
		return nil, fmt.Errorf("expected struct or struct ptr: got nil")
	}

	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct or struct ptr: %s", typ.Kind())
	}

	return typ, nil
}

// Initializer interface provides an Init function.
type Initializer interface {
	// Init initializes the command.
	Init(*App) error
}

// PlainCommand should be embedded in a struct to indicate that it implements a
// command. See package documentation for its usage.
type PlainCommand struct{}

// Execute is a placeholder for the function that runs the command.
func (c PlainCommand) Execute(args []string) error {
	return nil
}

// Command implements the default group flags. It is meant to be embedded into
// other application commands to provide default behavior for logging,
// profiling, etc.
type Command struct {
	PlainCommand
	LogOptions      `group:"Log Options"`
	ProfilerOptions `group:"Profiler Options"`
}

// Init implements initializer interface.
func (c Command) Init(a *App) error {
	if err := c.LogOptions.Init(a); err != nil {
		return err
	}

	if err := c.ProfilerOptions.Init(a); err != nil {
		return err
	}

	return nil
}
