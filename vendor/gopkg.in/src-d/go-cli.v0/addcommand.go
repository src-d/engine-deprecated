package cli

import (
	"fmt"
	"reflect"

	"github.com/jessevdk/go-flags"
)

// AddCommand adds a new command to the application. The command must have a
// special field defining name, short-description and long-description (see
// package documentation). It panics if the command is not valid.
// Returned CommandAdder can be used to add subcommands.
//
// Additional functions can be passed to manipulate the resulting *flags.Command
// after its initialization.
func (a *App) AddCommand(cmd interface{}, cfs ...func(*flags.Command)) CommandAdder {
	return commandAdder{a.Parser}.AddCommand(cmd, cfs...)
}

// CommandAdder can be used to add subcommands.
type CommandAdder interface {
	// AddCommand adds the given commands as subcommands of the
	// cuurent one.
	AddCommand(interface{}, ...func(*flags.Command)) CommandAdder
}

type commandAdder struct {
	internalCommandAdder
}

type internalCommandAdder interface {
	AddCommand(string, string, string, interface{}) (*flags.Command, error)
}

func (a commandAdder) AddCommand(cmd interface{}, cfs ...func(*flags.Command)) CommandAdder {
	typ, err := getStructType(cmd)
	if err != nil {
		panic(err)
	}

	pc, err := getPlainCommandField(typ)
	if err != nil {
		panic(err)
	}

	name := pc.Tag.Get("name")
	shortDescription := pc.Tag.Get("short-description")
	longDescription := pc.Tag.Get("long-description")

	c, err := a.internalCommandAdder.AddCommand(
		name, shortDescription, longDescription, cmd)
	if err != nil {
		panic(err)
	}

	for _, cf := range cfs {
		cf(c)
	}

	return commandAdder{c}
}

func getPlainCommandField(typ reflect.Type) (reflect.StructField, error) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if isPlainCommandField(field) {
			return field, nil
		}
	}

	return reflect.StructField{}, fmt.Errorf("PlainCommand not found")
}

func isPlainCommandField(field reflect.StructField) bool {
	if !field.Anonymous {
		return false
	}

	if field.Type == reflect.TypeOf(PlainCommand{}) {
		return true
	}

	for i := 0; i < field.Type.NumField(); i++ {
		if isPlainCommandField(field.Type.Field(i)) {
			return true
		}
	}

	return false
}
