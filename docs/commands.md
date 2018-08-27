# List of commands

This is a list of the commands that have been planned and whether
they've been implemented.

- [srcd init](#srcd-init)
- [srcd version](#srcd-version)
- [srcd parse](#srcd-parse)
    - [srcd parse uast](#srcd-parse-uast)
    - [srcd parse native](#srcd-parse-native)
    - [srcd parse lang](#srcd-parse-lang)
    - [srcd parse drivers](#srcd-parse-drivers)
        - [srcd parse drivers list](#srcd-parse-drivers-list)
        - [srcd parse drivers install](#srcd-parse-drivers-install)
        - [srcd parse drivers remove](#srcd-parse-drivers-remove)
        - [srcd parse drivers update](#srcd-parse-drivers-update)
- [srcd sql](#srcd-sql)
- [srcd web](#srcd-web)
- [srcd components](#srcd-components)
    - [srcd components status](#srcd-components-status)
    - [srcd components start](#srcd-components-start)
    - [srcd components stop](#srcd-components-stop)
    - [srcd components restart](#srcd-components-restart)
    - [srcd components install](#srcd-components-install)
    - [srcd components remove](#srcd-components-remove)
    - [srcd components update](#srcd-components-update)

## srcd
No action associated to this.

*flags*:
  * `-v|--verbose`: verbose mode on, log everything.

## srcd init
Initializes the `srcd` environment, starting (or restarting) the `srcd-server`
daemon, and verifying Docker is indeed installed and accessible.

It also records the what directory to use for later analysis with `gitbase`.
This will be either the given argument (only one accepted) or the current
directory if none is given.

*arguments*: working directory. If it's not provided, the current working directory will be used

*flags*: N/A

*status*: ✅ implemented

## srcd version
Shows the version of the current `srcd` cli binary, as well as the one for
the `srcd-server` running on Docker, and Docker itself.

*arguments*: N/A

*flags*: N/A

*status*: ✅ implemented

## srcd parse
All of the sub commands under `srcd parse` provide different kinds of parsing,
language classification, and bblfsh driver management.

*status*: ⛑ missing some

### srcd parse uast
Parses a file and returns the resulting UAST.
This command installs any missing drivers.

*arguments*:
  * `path`: file to be parsed, only one file supported at a time.

*flags*:
  * `-l|--lang`: skip language classification and force a specific language driver.
  * `-q|--query`: an XPath expression that will be applied on the obtained UAST.

*status*: ✅ done

### srcd parse native
Parses a file and returns the resulting native AST.
This command installs any missing drivers.

*arguments*:
  * `path`: file to be parsed, only one file supported at a time.

*flags*:
  * `-l|--lang`: skip language classification and force a specific language driver.

*status*: ⛔️ TBD

### srcd parse lang
Identifies the language of the given file.

*arguments*:

*flags*:

*status*: ✅ done

### srcd parse drivers
All of the subcomands of `srcd parse drivers` provide management for
the language drivers installed on `bblfsh`.

*status*: ⛔️ TBD

#### srcd parse drivers list
Lists all of the drivers already installed on `bblfsh` together with the
version installed.

*arguments*: N/A

*flags*: N/A

*status*: ✅ done

#### srcd parse drivers install
Installs the drivers for the given languages.

*arguments*: [language]* (the languages can have the following format `language` or `language:version`)

*status*: ✅ implemented

#### srcd parse drivers remove
Removes the drivers for the given languages.

*arguments*: [language]*

*status*: ✅ implemented

#### srcd parse drivers update
Updates the drivers for the given languages to the latest version or the one
indicated.

*arguments*: [language]* (the languages can have the following format `language` or `language:version`)

*status*: ✅ implemented

## srcd sql
Opens a sql client to a running `gitbase` server. If the server is not running,
it starts it automatically.

*arguments*: `query`: the query to run, if blank an interactive session is opened.

*flags*: N/A

*status*: ✅ implemented

## srcd web

All of the `web` subcommands provide web clients for different source{d} tools.

### srcd web parse

Opens a bblfsh web client.

*arguments*:

*flags*:
  * `--port`: port of the server

*status*: ✅ implemented

### srcd web sql

Opens a gitbase web client.

*arguments*:

*flags*:
  * `--port`: port of the server

*status*: ✅ implemented

## srcd components
The sub commands under `srcd components` provide management to pre-install,
remove, and update the components associated to the source{d} engine.

For instance, `bblfsh` and `gitbase` are some of these components.
More will be coming soon. One of them could easily be the Spark engine with
Jupyter.

*status*: ⛔️ TBD (not necessary for alpha)

### srcd components status
TBD

### srcd components start
TBD

### srcd components stop
TBD

### srcd components restart
TBD

### srcd components install
TBD

### srcd components remove
TBD

### srcd components update
TBD
