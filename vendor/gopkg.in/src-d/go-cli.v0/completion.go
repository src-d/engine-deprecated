package cli

import (
	"bytes"
	"os"
	"text/template"

	"github.com/jessevdk/go-flags"
)

// CompletionCommand defines the default completion command. Most of the time, it
// should not be used directly, since it will be added by default to the App.
type CompletionCommand struct {
	PlainCommand `name:"completion" short-description:"print bash completion script"`
	Name         string
}

// InitCompletionCommand returns an additional AddCommand function that fills
// the CompletionCommand long description
func InitCompletionCommand(appname string) func(*flags.Command) {
	return func(c *flags.Command) {
		t := template.Must(template.New("desc").Parse(
			`Print a bash completion script for {{.Name}}.

You can place it on /etc/bash_completion.d/{{.Name}}, or add it to your .bashrc:
    echo "source <({{.Name}} completion)" >> ~/.bashrc
`))

		var tpl bytes.Buffer
		t.Execute(&tpl, struct{ Name string }{appname})
		c.LongDescription = tpl.String()
	}
}

// Execute runs the install command.
func (c CompletionCommand) Execute(args []string) error {
	t := template.Must(template.New("completion").Parse(
		`# Save this file to /etc/bash_completion.d/{{.Name}}
#
# or add the following line to your .bashrc file: 
#   echo "source <({{.Name}} completion)" >> ~/.bashrc

_completion-{{.Name}}() {
    # All arguments except the first one
    args=("${COMP_WORDS[@]:1:$COMP_CWORD}")

    # Only split on newlines
    local IFS=$'\n'

    # Call completion (note that the first element of COMP_WORDS is
    # the executable itself)
    COMPREPLY=($(GO_FLAGS_COMPLETION=1 ${COMP_WORDS[0]} "${args[@]}"))
    return 0
}

complete -F _completion-{{.Name}} {{.Name}}
`))

	err := t.Execute(os.Stdout, struct{ Name string }{c.Name})
	if err != nil {
		return err
	}

	return nil
}
