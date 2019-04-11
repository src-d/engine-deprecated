# List of commands

This is a list of the commands that have been planned and whether
they've been implemented.

- [srcd init](#srcd-init)
- [srcd stop](#srcd-stop)
- [srcd prune](#srcd-prune)
- [srcd version](#srcd-version)
- [srcd parse](#srcd-parse)
    - [srcd parse uast](#srcd-parse-uast)
    - [srcd parse lang](#srcd-parse-lang)
    - [srcd parse drivers](#srcd-parse-drivers)
- [srcd sql](#srcd-sql)
- [srcd web](#srcd-web)
    - [srcd web parse](#srcd-web-parse)
    - [srcd web sql](#srcd-web-sql)
- [srcd components](#srcd-components)
    - [srcd components list](#srcd-components-list)
    - [srcd components install](#srcd-components-install)
    - [srcd components start](#srcd-components-start)

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

## srcd stop

Stops all containers used by the source{d} Engine.

*arguments*: N/A

*flags*: N/A

## srcd prune

Removes all containers and docker volumes used by the source{d} engine.

*arguments*: N/A

*flags*:
  * `--with-images`: remove docker images too

## srcd version
Shows the version of the current `srcd` cli binary, as well as the one for
the `srcd-server` running on Docker, and Docker itself.

*arguments*: N/A

*flags*: N/A

## srcd parse
All of the sub commands under `srcd parse` provide different kinds of parsing,
language classification, and bblfsh driver management.

### srcd parse uast
Parses a file and returns the resulting UAST.

*arguments*:
  * `path`: file to be parsed, only one file supported at a time.

*flags*:
  * `-l|--lang`: skip language classification and force a specific language driver.
  * `-q|--query`: an XPath expression that will be applied on the obtained UAST.
  * `-m|--mode`: UAST parsing mode: semantic|annotated|native (default "semantic")

### srcd parse lang
Identifies the language of the given file.

*arguments*:

*flags*:

### srcd parse drivers
Lists all of the drivers already installed on `bblfsh` together with the
version installed.

*arguments*: N/A

*flags*: N/A

## srcd sql
Opens a sql client to a running `gitbase` server. If the server is not running,
it starts it automatically.

*arguments*: `query`: the query to run, if blank an interactive session is opened.

*flags*: N/A

## srcd web

All of the `web` subcommands provide web clients for different source{d} tools.

### srcd web parse

Opens a bblfsh web client.

*arguments*:

### srcd web sql

Opens a gitbase web client.

*arguments*:

## srcd components
The sub commands under `srcd components` provide management to pre-install,
remove, and update the components associated to the source{d} Engine.

For instance, `bblfsh` and `gitbase` are some of these components.
More will be coming soon. One of them could easily be the Spark engine with
Jupyter.


### srcd components list

Lists source{d} Engine components

*arguments*:

*flags*:
  * `-a|--all`: show all versions found

### srcd components install

Installs source{d} Engine components images.

*arguments*:
  * `component`: the name of the component image. It must be one of:
    * `bblfsh/bblfshd`
    * `bblfsh/web`
    * `srcd/cli-daemon`
    * `srcd/gitbase-web`
    * `srcd/gitbase`

*flags*: N/A

### srcd components status

*status*: ❌ TBD

### srcd components start

Start source{d} Engine components and its dependencies if needed.

*arguments*:
  * `component`: the name of the component image. It must be one of:
    * `bblfsh/bblfshd`
    * `bblfsh/web`
    * `srcd/cli-daemon`
    * `srcd/gitbase-web`
    * `srcd/gitbase`

*flags*: N/A

### srcd components stop

*status*: ❌ TBD

### srcd components restart

*status*: ❌ TBD

### srcd components remove

*status*: ❌ TBD

### srcd components update

*status*: ❌ TBD
