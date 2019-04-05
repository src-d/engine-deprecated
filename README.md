<a href="https://www.sourced.tech/engine">
  <img src="docs/sourced-engine.png" alt="source{d} Engine" height="120px" />
</a>

**Powerful language-agnostic analysis of your source code and git history.**

[![GitHub version](https://badge.fury.io/gh/src-d%2Fengine.svg)](https://github.com/src-d/engine/releases)
[![Build Status](https://travis-ci.com/src-d/engine.svg?branch=master)](https://travis-ci.com/src-d/engine)
[![Go Report Card](https://goreportcard.com/badge/github.com/src-d/engine)](https://goreportcard.com/report/github.com/src-d/engine)
[![GoDoc](https://godoc.org/github.com/src-d/engine?status.svg)](https://godoc.org/github.com/src-d/engine)
[![source{d} design document](https://img.shields.io/badge/source%7Bd%7D-design%20document-blue.svg)](https://drive.google.com/open?id=1Fw-qE8mC3lZUP5hoH1Jo5bYcGV0Z018uAq7er3oAMLw)

[Website](https://www.sourced.tech/engine) •
[Documentation](https://docs.sourced.tech/engine) •
[Blog](https://blog.sourced.tech) •
[Slack](http://bit.ly/src-d-community) •
[Twitter](https://twitter.com/sourcedtech)

## Introduction

source{d} Engine exposes powerful [Universal ASTs](#babelfish-uast) to analyze your code and a SQL engine to analyze your git history:

- **Code Processing**: use git repositories as a dataset.
- **Language-Agnostic Code Analysis**: automatically identify languages, parse source code, and extract the pieces that matter with language-independent queries.
- **Git Analysis**: powerful SQL based analysis on top of your git repositories.
- **Querying With Familiar APIs**: analyze your code through powerful friendly APIs, such as SQL, gRPC, and various client libraries.

You can access a rendered version of this documentation at [docs.sourced.tech/engine](https://docs.sourced.tech/engine).

## Contents

- [Quick Start](#quick-start)
  - [Requirements](#1-install-docker)
  - [Installation](#2-install-source-d-engine)
  - [Initialization](#3-start-source-d-engine-with-your-local-repositories)
  - [Commands](#4-explore-source-d-engine)
  - [Examples](#5-start-executing-queries)
- [Other Guides & Examples](#other-guides-and-examples)
- [Architecture](#architecture)
- [Babelfish UAST](#babelfish-uast)
- [Clients & Connectors](#clients-and-connectors)
- [Community](#community)
- [Contributing](#contributing)
- [Credits](#credits)
- [License](#license)

## Quick Start

Follow the steps below to get started with source{d} Engine.

### 1. Install Docker

Follow these instructions based on your OS:

#### Docker on macOS

Follow instructions at [Docker for Mac](https://store.docker.com/editions/community/docker-ce-desktop-mac). You may also use [Homebrew](https://brew.sh/):

  ```bash
  brew cask install docker
  ```

#### Docker on Ubuntu Linux

Follow instructions at [Docker for Ubuntu Linux](https://docs.docker.com/install/linux/docker-ce/ubuntu/#install-docker-ce-1):

  ```bash
  sudo apt-get update
  sudo apt-get install docker-ce
  ```

#### Docker on Arch Linux

Follow instructions at [Docker for Arch Linux](https://wiki.archlinux.org/index.php/Docker#Installation):

  ```bash
  sudo pacman -S docker
  ```
- [Docker Desktop for Windows](https://hub.docker.com/editions/community/docker-ce-desktop-windows). Make sure to read the system requirements [here](https://docs.docker.com/docker-for-windows/install/). Please note Docker Toolbox is not supported.

### 2. Install source{d} Engine

Download the **[latest release](https://github.com/src-d/engine/releases/latest)** for MacOS (Darwin), Linux or Windows.

#### Engine on macOS

Double-click on the tar file to extract it.

Open your terminal and move the binary to your local bin folder to make it executable from any directory:

```bash
sudo mv ~/replace/path/to/engine_darwin_amd64/srcd /usr/local/bin/
```

#### Engine on Linux

Extract the contents of the tar file from your terminal:

```bash
tar -xvf ~/replace/path/to/engine_REPLACEVERSION_linux_amd64.tar.gz
```

Move the binary to your local bin folder to be executable from any directory:

```bash
sudo mv engine_linux_amd64/srcd /usr/local/bin/
```

#### Engine on Windows

*Please note that from now on we assume that the commands are executed in `powershell` and not in `cmd`. Running them in `cmd` is not guaranteed to work. Proper support may be added in future releases.*

To run the following preparatory steps you need to run powershell as administrator.

```powershell
mkdir 'C:\Program Files\srcd'
# Add the directory to the `%path%` to make it available from anywhere
setx /M PATH "$($env:path);C:\Program Files\srcd"
# Now open a new powershell to apply the changes
```

Extract the tar file with the tool you prefer. Copy `srcd.exe` into the directory you created:
```powershell
mv engine_windows_amd64\srcd.exe 'C:\Program Files\srcd'
```

### 3. Start source{d} Engine With Your Local Repositories

Now it's time to initialize source{d} Engine and provide it with some repositories to analyze:

```bash
# Without a path Engine operates on the local directory,
# it works with nested or sub-directories.
srcd init

# You can also provide a path
srcd init <path>
```

**Note:**
Once Engine is initialized with a working dir, it does not watch for new repository creation, so if you want to add (or delete) repositories, you need to `init` again.
Also, database indexes are not updated automatically when its contents change, so in that cases, the index must be manually recreated.
If you want to inspect different datasets, it needs to be done separatelly: you can `init` one working directory, perform your queries, and then change to the other dataset performing an `init` with the working directory containing it. In such situations, indexes are kept, so you can change from one dataset to the other, without having to recreate them.

**Note for MacOS:**
Docker for Mac [requires enabling file sharing](https://docs.docker.com/docker-for-mac/troubleshoot/#volume-mounting-requires-file-sharing-for-any-project-directories-outside-of-users) for any path outside of `/Users`.

### 4. Explore source{d} Engine

_For the full list of the commands supported by `srcd` and those
that have been planned, please read [commands.md](docs/commands.md)._

**Note for Windows:** Docker for Windows [requires shared drives](https://docs.docker.com/docker-for-windows/#shared-drives). Other than that, it's important to use a workdir that doesn't include any sub-directory whose access is not readable by the user running `srcd`. As an example using `C:\Users` as workdir will most probably not work. For more details see [this issue](https://github.com/src-d/engine/issues/250).

source{d} Engine provides interfaces to [query code repositories](#querying-code) and to [parse code](#parsing-code) into [Universal Abstract Syntax Trees](#babelfish-uast).

In this section we will cover a mix of some commands and interfaces available.

**Note:**
source{d} Engine will download and install Docker images on demand. Therefore, the first time you run some of these commands, they might take a bit of time to start up. Subsequent runs will be faster.

#### Querying Code

##### Query Web Interface

To launch the [web client for the SQL interface](https://github.com/src-d/gitbase-web), run the following command and start executing queries:

```bash
# Launch the query web client
srcd web sql
```

This should open the [web interface](https://github.com/src-d/gitbase-web) in your browser.
You can also access it directly at [http://localhost:8080](http://localhost:8080).

##### Query Command Line Interface (CLI)

If you prefer to work within your terminal via command line, you can open a SQL REPL
that allows you to execute queries against your repositories by executing:

```bash
# Launch the query CLI REPL
srcd sql
```

If you want to run a query directly, you can also execute it as such:

```bash
# Run query via CLI
srcd sql "SHOW tables;"
```

**Note:**
Engine's SQL supports a [UAST](#babelfish-uast) function that returns a Universal AST for the selected source text. UAST values are returned as binary blobs, and are best visualized in the `web sql` interface rather than the CLI where are seen as binary data.

#### Parsing Code

Sometimes you may want to parse files directly as [UASTs](#babelfish-uast).

To see which languages are available, check the table of [supported languages](#babelfish-uast).

##### Parse Web Client

If you want a playground to see examples of the UAST, or run your own, you can launch the [parse web client](https://github.com/bblfsh/web).


```bash
# Launch the parse web client
srcd web parse
```

This should open the [web interface](https://github.com/bblfsh/web) in your browser.
You can also access it directly at [http://localhost:8081](http://localhost:8081).

##### Parse Command Line Interface (CLI)

Alternatively, you can also start parsing files on the command line:

```bash
# Parse file via CLI
srcd parse uast /path/to/file.java
```

To parse a file specifying the programming language:

```bash
srcd parse uast --lang=LANGUAGE /path/to/file
```

To see the installed language drivers:

```bash
srcd parse drivers list
```

### 5. Start Executing Queries

**Understand which tables are available for you to query:**

```bash
gitbase> show tables;
+--------------+
|    TABLE     |
+--------------+
| blobs        |
| commit_blobs |
| commit_files |
| commit_trees |
| commits      |
| files        |
| ref_commits  |
| refs         |
| remotes      |
| repositories |
| tree_entries |
+--------------+
```

```bash
gitbase> DESCRIBE TABLE commits;
+---------------------+-----------+
|        NAME         |   TYPE    |
+---------------------+-----------+
| repository_id       | TEXT      |
| commit_hash         | TEXT      |
| commit_author_name  | TEXT      |
| commit_author_email | TEXT      |
| commit_author_when  | TIMESTAMP |
| committer_name      | TEXT      |
| committer_email     | TEXT      |
| committer_when      | TIMESTAMP |
| commit_message      | TEXT      |
| tree_hash           | TEXT      |
| commit_parents      | JSON      |
+---------------------+-----------+
```

**Show me the repositories I am analyzing:**

```sql
SELECT * FROM repositories;
```

**Top 10 repositories by commit count in [HEAD](https://git-scm.com/book/en/v2/Git-Internals-Git-References#ref_the_ref):**

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
LIMIT 10;
```

**Query all files from [HEAD](https://git-scm.com/book/en/v2/Git-Internals-Git-References#ref_the_ref):**

```sql
SELECT cf.file_path, f.blob_content
FROM ref_commits r
NATURAL JOIN commit_files cf
NATURAL JOIN files f
WHERE r.ref_name = 'HEAD'
AND r.history_index = 0;
```

**Retrieve the UAST for all files from [HEAD](https://git-scm.com/book/en/v2/Git-Internals-Git-References#ref_the_ref):**

```sql
SELECT * FROM (
    SELECT cf.file_path,
           UAST(f.blob_content, LANGUAGE(f.file_path,  f.blob_content)) as uast
    FROM ref_commits r
    NATURAL JOIN commit_files cf
    NATURAL JOIN files f
    WHERE r.ref_name = 'HEAD'
    AND r.history_index = 0
) t WHERE uast != '';
```

**Query for all LICENSE & README files across history:**

```sql
SELECT repository_id, blob_content
FROM files
WHERE file_path = 'LICENSE'
OR file_path = 'README.md';
```

You can find further sample queries in the [examples](examples/README.md) folder.

<!---
### 6. Next steps

You can now run source{d} Engine, choose what you would like to do next:

- [**Analyze your git repositories**](#)
- [**Understand how your code has evolved**](#)
- [**Write your own static analysis rules**](#)
- [**Build a data pipeline for MLonCode**](#)
--->

## Other Guides and Examples

Here is a collection of documentation, guides, and examples of the components exposed by source{d} Engine:

- [gitbase documentation](https://docs.sourced.tech/gitbase/): table schemas, syntax, functions, examples.
- [Babelfish documentation](https://docs.sourced.tech/babelfish/): specifications, usage, examples.

## Architecture

source{d} Engine functions as a command-line interface tool that provides easy access to components of source{d} stack for Code As Data.

It consists of a daemon managing all of the services, which are packaged as Docker containers:

- [enry](https://github.com/src-d/enry): language classifier
- [babelfish](https://doc.bblf.sh): universal code parser
  - [daemon](https://github.com/bblfsh/bblfshd): Babelfish server
  - [language drivers](https://github.com/search?q=topic%3Adriver+org%3Abblfsh&type=Repositories): parsers + normalizers for programming languages
  - [babelfish-web](https://github.com/bblfsh/web): web client for Babelfish server
- [gitbase](https://github.com/src-d/gitbase): SQL database interface to Git repositories
  - [gitbase-web](https://github.com/src-d/gitbase-web): web client for gitbase

For more details on the architecture of this project, read [docs/architecture.md](docs/architecture.md).

![diagram of source{d} Engine](docs/architecture-diagram.png)


## Babelfish UAST

### Definition

One of the most important components of source{d} Engine is the UAST, which stands for:
[Universal Abstract Syntax Tree](https://docs.sourced.tech/babelfish/uast/uast-specification).

UASTs are a normalized form of a programming language's AST, annotated with language-agnostic roles and transformed with language-agnostic concepts (e.g. Functions, Imports etc.).

These enable advanced static analysis of code and easy feature extraction for statistics or [Machine Learning on Code](https://github.com/src-d/awesome-machine-learning-on-source-code).

### UAST Usage

To parse a file for a UAST using source{d} Engine, head to the [Parsing Code section](#parsing-code) of this document.

### Supported Languages

To see which languages are available, check the table of [Babelfish supported languages](https://docs.sourced.tech/babelfish/languages).

## Clients and Connectors

For connecting to the language parsing server (Babelfish) and analyzing the UAST, there are several language clients currently supported and maintained:

- [Babelfish Go Client](https://github.com/bblfsh/client-go)
- [Babelfish Python Client](https://github.com/bblfsh/client-python)
- [Babelfish Scala Client](https://github.com/bblfsh/client-scala)

The Gitbase Spark connector is under development, which aims to allow for an easy integration with Spark & PySpark:

- [Gitbase Spark Connector](https://github.com/src-d/gitbase-spark-connector) - coming soon!

## Community

source{d} has an amazing community of developers and contributors who are interested in Code As Data and/or Machine Learning on Code. Please join us! 👋

- [Community](https://sourced.tech/community/)
- [Slack](http://bit.ly/src-d-community)
- [Twitter](https://twitter.com/sourcedtech)
- [Email](mailto:hello@sourced.tech)

## Contributing

Contributions are **welcome and very much appreciated** 🙌
Please refer [to our Contribution Guide](CONTRIBUTING.md) for more details.

## Credits

This software uses code from open source packages. We'd like to thank the contributors for all their efforts:

- [Cobra](https://github.com/spf13/cobra)

## License

[Apache 2.0 License](LICENSE)
