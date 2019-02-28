# List of commands

This is a list of the commands that have been planned and whether
they've been implemented.

- [srcd init](#srcd-init)
- [srcd version](#srcd-version)
- [srcd parse](#srcd-parse)
    - [srcd parse uast](#srcd-parse-uast)
    - [srcd parse lang](#srcd-parse-lang)
    - [srcd parse drivers](#srcd-parse-drivers)
        - [srcd parse drivers list](#srcd-parse-drivers-list)
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

*global flags for all sub commands*:
  * `-v|--verbose`: verbose mode on, log everything.
  * `--config`: path to the config file.

The config file is optional. By default `srcd` will look for it in `$HOME/.srcd/config.yml`. You can use a YAML file to configure the public port bindings of the components containers.

Example config file with the default values:

```yaml
# Any change in the exposed ports will require you to run srcd init (or stop)

components:
  bblfshd:
    port: 9432

  bblfsh_web:
    port: 8081

  gitbase_web:
    port: 8080

  gitbase:
    port: 3306

  daemon:
    port: 4242
```

## srcd init
Initializes the `srcd` environment, starting (or restarting) the `srcd-server`
daemon, and verifying Docker is indeed installed and accessible.

It also records the what directory to use for later analysis with `gitbase`.
This will be either the given argument (only one accepted) or the current
directory if none is given.

*arguments*: working directory. If it's not provided, the current working directory will be used

*flags*: N/A

*status*: ✅ implemented

## srcd stop

Stops all containers used by the source{d} engine.

*arguments*: N/A

*flags*: N/A

*status*: ✅ implemented

## srcd prune

Removes all containers and docker volumes used by the source{d} engine.

*arguments*: N/A

*flags*:
  * `--with-images`: remove docker images too

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

*status*: ✅ implemented

### srcd parse uast
Parses a file and returns the resulting UAST.
This command installs any missing drivers.

*arguments*:
  * `path`: file to be parsed, only one file supported at a time.

*flags*:
  * `-l|--lang`: skip language classification and force a specific language driver.
  * `-q|--query`: an XPath expression that will be applied on the obtained UAST.
  * `-m|--mode`: UAST parsing mode: semantic|annotated|native (default "semantic")

*status*: ✅ done

### srcd parse lang
Identifies the language of the given file.

*arguments*:

*flags*:

*status*: ✅ done

### srcd parse drivers
All of the subcomands of `srcd parse drivers` provide management for
the language drivers installed on `bblfsh`.

*status*: ✅ implemented

#### srcd parse drivers list
Lists all of the drivers already installed on `bblfsh` together with the
version installed.

*arguments*: N/A

*flags*: N/A

*status*: ✅ done

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

*status*: ✅ implemented

### srcd web sql

Opens a gitbase web client.

*arguments*:

*status*: ✅ implemented

## srcd components
The sub commands under `srcd components` provide management to pre-install,
remove, and update the components associated to the source{d} engine.

For instance, `bblfsh` and `gitbase` are some of these components.
More will be coming soon. One of them could easily be the Spark engine with
Jupyter.

*status*: ⛔️ TBD (not necessary for alpha)

### srcd components list

Lists source{d} components

*arguments*:

*flags*:
  * `-a|--all`: show all versions found

*status*: ✅ implemented

### srcd components install

Installs source{d} components images.

*arguments*:
  * `component`: the name of the component image. It must be one of:
    * `bblfsh/bblfshd`
    * `bblfsh/web`
    * `srcd/cli-daemon`
    * `srcd/gitbase-web`
    * `srcd/gitbase`

*flags*: N/A

### srcd components status
TBD

### srcd components start
TBD

### srcd components stop
TBD

### srcd components restart
TBD

### srcd components remove
TBD

### srcd components update
TBD
