<h1 align="center">
  <br>
  <a href="https://www.sourced.tech"><img src="docs/sourced.png" alt="source{d}" height="60px"></a>
  <br>
  <br>
  source{d} Engine
  <br>
</h1>

<h3 align="center">Powerful language-agnostic analysis of your source code and git history.</h3>

<p align="center">
  <a href="https://github.com/src-d/engine/releases">
    <img src="https://badge.fury.io/gh/src-d%2Fengine.svg"
         alt="GitHub version">
  </a>
  <a href="https://travis-ci.org/src-d/engine">
    <img src="https://travis-ci.org/src-d/engine.svg?branch=master"
         alt="Build Status">
  </a>
  <a href="https://codecov.io/gh/src-d/engine">
    <img src="https://codecov.io/gh/src-d/engine/branch/master/graph/badge.svg"
         alt="codecov">
  </a>  
  <a href="https://goreportcard.com/report/github.com/src-d/engine">
    <img src="https://goreportcard.com/badge/github.com/src-d/engine"
         alt="Go Report Card">
  </a>   
  <a href="https://godoc.org/gopkg.in/src-d/engine.v0">
    <img src="https://godoc.org/gopkg.in/src-d/engine.v0?status.svg"
         alt="GoDoc">
  </a>
  <a href="https://drive.google.com/open?id=1Fw-qE8mC3lZUP5hoH1Jo5bYcGV0Z018uAq7er3oAMLw">
    <img src="https://img.shields.io/badge/source%7Bd%7D-design%20document-blue.svg"
         alt="source{d} design document">
  </a>
</p>

<p align="center"><b>
    <a href="https://www.sourced.tech">Website</a> •  
    <a href="https://docs.sourced.tech">Documentation</a> •
    <a href="https://blog.sourced.tech">Blog</a> •
    <a href="http://bit.ly/src-d-community">Slack</a> •
    <a href="https://twitter.com/sourcedtech">Twitter</a>
</b></p>

## Introduction

source{d} Engine exposes powerful Universal AST's to analyze your code and a SQL engine to analyze your git history:

- **Code Retrieval**: retrieve and store git repositories as a dataset.
- **Language Agnostic Code Analysis**: automatically identify languages, parse source code, and extract the pieces that matter in a completely language-agnostic way.
- **Git Analysis** powerful SQL based analysis on top of your git repositories.
- **Querying With Familiar APIs** analyze your code through powerful friendly APIs, such as SQL, gRPC, REST, and various client libraries.

## Contents

- [Quickstart](#quickstart)
- [Guides & Examples](#guides-and-examples)
- [Architecture](#architecture)
- [Babelfish UAST](#babelfish-uast)
- [Clients & Connectors](#clients-connectors)
- [Community](#community)
- [Contributing](#contributing)
- [Credits](#credits)
- [License](#license)

## Quickstart

Follow the steps below to get started with source{d| Engine.

#### 1. Install Docker

Follow these instructions:

- [Docker for Mac](https://store.docker.com/editions/community/docker-ce-desktop-mac)
- [Docker for Windows](https://store.docker.com/editions/community/docker-ce-desktop-windows)
- [Docker for Ubuntu](https://docs.docker.com/install/linux/docker-ce/ubuntu/#install-docker-ce-1)
```bash
sudo apt-get update
sudo apt-get install docker-ce
```
- [Docker for Arch Linux](https://wiki.archlinux.org/index.php/Docker#Installation)
```bash
sudo pacman -S docker
```

#### 2. Install source{d} Engine

Download the [latest release](https://github.com/src-d/engine/releases) for MacOS (Darwin), Linux or Windows.

MacOS / Linux:

```
# Make it executable
chmod +ux srcd
# Move it into your local bin folder to be executable from anywhere
sudo mv srcd /usr/local/bin/
```

#### 3. Start source{d} Engine with your local repositories

Now it's time to initialize the source{d} engine and provide it some repositories to analyze:

```
# Without a path it operates on the local folder,
# it works with nested folders.
srcd init

# You can also provide a path
srcd init /home/user/replace/path/
```

#### 4. Explore the source{d} Engine

To launch the web client, run the following command and start executing queries:

```bash
srcd web sql
```

In your browser, now go to http://localhost:8080

If you prefer to stay with the command line, you can execute:

```bash
srcd sql
```

This will open a SQL client that allows you to execute queries against your repositories.

If you want to run a query directly, you can also execute it as such:

```bash
srcd sql "SHOW tables;"
```

#### 5. Start executing queries

**Top 10 repositories by commit count in HEAD**:

```sql
SELECT repository_id,commit_count 
FROM (
    SELECT r.repository_id, COUNT(*) AS commit_count
    FROM ref_commits r
    WHERE r.ref_name = 'HEAD'
    GROUP BY r.repository_id
) AS q
ORDER BY commit_count
DESC
LIMIT 10
```

**Query all files from HEAD**:

```sql
SELECT cf.file_path, f.blob_content 
FROM ref_commits r 
NATURAL JOIN commit_files cf 
NATURAL JOIN files f 
WHERE r.ref_name = 'HEAD' 
AND r.index = 0
```

**Retrieve the UAST for all files from HEAD**:

```sql
SELECT * FROM (
    SELECT cf.file_path,
           UAST(f.blob_content, LANGUAGE(f.file_path,  f.blob_content)) as uast
    FROM ref_commits r 
    NATURAL JOIN commit_files cf 
    NATURAL JOIN files f 
    WHERE r.ref_name = 'HEAD' 
    AND r.index = 0
) t WHERE ARRAY_LENGTH(uast) > 0
```

**Query for all LICENSE & README files across history**:

```sql
SELECT repository_id, blob_content 
FROM files 
WHERE file_path = 'LICENSE' 
OR file_path = 'README.md'
```

You can find further sample queries in the [examples](examples/README.md) folder.

#### 6. Next steps

You can now run the source{d} Engine, choose what you would like to do next:

- [**Analyze your git repositories**](#)
- [**Understand how your code has evolved**](#)
- [**Write your own static analysis rules**](#)
- [**Build a data pipeline for MLonCode**](#)

## Guides and Examples

For the full list of the commands supported by `srcd` and those
that have been planned, please read [commands.md](docs/commands.md).

Collection of guide & examples using the source{d} Engine:

- [SonarSource Java Static Analysis Rules using Babelfish](https://github.com/bblfsh/sonar-checks)


## Architecture

source{d} Engine functions as a CLI tool that provides easy access to components of the source{d} stack for Code As Data. It consists of a daemon managing all of the services (Babelfish, Enry, Gitbase etc.) which are packaged as docker containers.

<p align="center"><img src="docs/architecture.png" height="150" /></p>

For more details on the architecture of this project, read [docs/architecture.md](docs/architecture.md).

## Babelfish UAST

One of the most important components of the source{d} engine is the UAST. 

UAST stands for [Universal Abstract Syntax Tree](https://docs.sourced.tech/babelfish/uast/uast-specification), it is a normalized form of a programming language's AST, annotated with language agnostic roles and transformed with language agnostic concepts (e.g. Functions, Imports etc.). It enables advanced static analysis of code and easy feature extraction for statistics or Machine Learning on Code.

To parse a file for a UAST, it is as easy as:

```bash
srcd parse uast --lang=LANGUAGE /path/to/file
```

To launch the web client, run the following command and start executing queries:

```bash
srcd web parse
```

In your browser, now go to http://localhost:8081

## Clients & Connectors

For connecting to the language parsing server (Babelfish) and analyzing the UAST, there are several language clients currently supported and maintained:

- [Babelfish Go Client](https://github.com/bblfsh/client-go)
- [Babelfish Python Client](https://github.com/bblfsh/client-python)
- [Babelfish Scala Client](https://github.com/bblfsh/client-scala)

The Gitbase Spark connector is under development, which aims to allow for an easy integration with Spark & PySpark:

- [Gitbase Spark Connector](https://github.com/src-d/gitbase-spark-connector)

## Community

source{d} has an amazing community of developers & contributors who are interested in Code As Data and/or Machine Learning on Code. Please join us! 👋

- [Slack](http://bit.ly/src-d-community)
- [Twitter](https://twitter.com/sourcedtech)
- [Email](mailto:hello@sourced.tech)

## Contributing

Contributions are **welcome and very much appreciated** 🙌
Please refer [to our contribution guide](CONTRIBUTING.md) for more details.

## Credits

This software uses code from several open source packages. We'd like to thank the contributors for all their efforts:

- [Cobra](https://github.com/spf13/cobra)

## License

[Apache 2.0 License](LICENSE)
