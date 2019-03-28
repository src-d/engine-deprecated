# Changelog

## [Unreleased]
<details>
  <summary>
    Changes that have landed in master but are not yet released.
    Click to see more.
  </summary>

### Breaking Changes

In this release we have changed how Engine stores the [gitbase index](https://docs.sourced.tech/gitbase/using-gitbase/indexes) data. In previous releases we stored this data in `~/.srcd/gitbase`. From now on they will be managed as docker volumes.

This change has 2 implications:

- If you had created any index for your data you will need to recreate them after source{d} Engine is updated.
- The files in `~/.srcd/gitbase` are not used anymore, and you need to clean them manually:

  ```bash
  rm -rf ~/.srcd/gitbase
  ```
  Please not that in Linux you may need to run the command as root.

### Components

- `srcd/gitbase-web` has been updated to [v0.6.4](https://github.com/src-d/gitbase-web/releases/tag/v0.6.4).
- `bblfsh/bblfshd` has been updated to [v2.12.0-drivers](https://github.com/bblfsh/bblfshd/releases/tag/v2.12.0).

### New Features

- `srcd prune` now removes the gitbase index data ([#352](https://github.com/src-d/engine/issues/352)).
- More friendlier and useful error messages ([#252](https://github.com/src-d/engine/issues/252), [#258](https://github.com/src-d/engine/issues/258), [#272](https://github.com/src-d/engine/issues/272), [#291](https://github.com/src-d/engine/issues/291), [#294](https://github.com/src-d/engine/issues/294), [#295](https://github.com/src-d/engine/issues/295), [#326](https://github.com/src-d/engine/issues/326)).
- Replace the basic REPL SQL shell with a standard MySQL client ([#154](https://github.com/src-d/engine/issues/154)).
- Show the container exposed ports in the `srcd components list` output ([#300](https://github.com/src-d/engine/issues/300)).
- Set `delegated` consistency mode for mounted volumes on macOS. This improves SQL queries performance ([#330](https://github.com/src-d/engine/issues/330)).
- Use volumes for gitbase index data persistence ([#373](https://github.com/src-d/engine/issues/373)).

### Bug Fixes

- Fix the gRCP `ResourceExhausted` error for `srcd parse uast` with big files ([#271](https://github.com/src-d/engine/issues/271)).
- In cases where the workdir contains a lot of Git repositories, `srcd web sql` could open the web UI before gitbase was ready to accept queries ([#284](https://github.com/src-d/engine/issues/284)).

</details>

## [v0.11.0](https://github.com/src-d/engine/releases/tag/v0.11.0) - 2019-03-08

### Components

- `srcd/gitbase` has been updated to [v0.19.0](https://github.com/src-d/gitbase/releases/tag/v0.19.0)
- `srcd/gitbase-web` has been updated to [v0.6.2](https://github.com/src-d/gitbase-web/releases/tag/v0.6.2).
- `bblfsh/bblfshd` has been updated to [v2.11.8-drivers](https://github.com/bblfsh/bblfshd/releases/tag/v2.11.8).

### Windows Support

This release brings back windows binaries. source{d} Engine has been tested on Windows 10 ([#235](https://github.com/src-d/engine/issues/235)).

### New Features

You can now manage the public ports of the components containers, using a YAML config file. This allows you to avoid port conflicts with other services that may be using the default ports. For more information see the [CLI documentation](https://docs.sourced.tech/engine/learn-more/commands#srcd) ([#236](https://github.com/src-d/engine/issues/236)).

### Known Issues

- [#297](https://github.com/src-d/engine/issues/297): `srcd parse` does not detect the language automatically for C#, C++, or bash files. For these languages you will need to set `--lang` manually. For example: 
```
$ srcd parse uast file.cs --lang csharp
$ srcd parse uast file.cpp --lang cpp
$ srcd parse uast file.bash --lang bash
```

- [Windows only] [#349](https://github.com/src-d/engine/issues/349): Engine cannot handle gitbase indexes.
- [Windows only] [#257](https://github.com/src-d/engine/issues/257): `srcd sql` REPL prints unix terminal control characters.

## [v0.10.0](https://github.com/src-d/engine/releases/tag/v0.10.0) - 2019-02-22

### Components

- `srcd/gitbase-web` has been updated to [v0.6.0](https://github.com/src-d/gitbase-web/releases/tag/v0.6.0).
- `bblfsh/bblfshd` has been updated to [v2.11.7-drivers](https://github.com/bblfsh/bblfshd/releases/tag/v2.11.7).

### Windows Support

This release does not include windows binaries. We are working on ensuring windows is properly supported and it will be included in the next releases.

### Bug Fixes

- Fix `connection refused` errors when gitbase takes time to process the working directory repositories. Now engine waits until it is ready, showing a nice spinner ([#195](https://github.com/src-d/engine/issues/195), [#216](https://github.com/src-d/engine/issues/216)).
- Fix error message `error while marshaling: proto: invalid UTF-8 string` for SQL queries including UAST columns ([#196](https://github.com/src-d/engine/issues/196)).
- Use higher timeouts when starting and stopping containers ([#207](https://github.com/src-d/engine/issues/207), [#213](https://github.com/src-d/engine/issues/213)).
- Add the optional working directory argument in the output of `srcd init -h` ([#203](https://github.com/src-d/engine/issues/203)).

## [v0.8.0](https://github.com/src-d/engine/releases/tag/v0.8.0) - 2019-01-22

### Components

- `srcd/gitbase-web` has been updated to [v0.5.0](https://github.com/src-d/gitbase-web/releases/tag/v0.5.0).
- `bblfsh/web` has been updated to [v0.9.0](https://github.com/bblfsh/web/releases/tag/v0.9.0).

### New Features

- `srcd sql` now installs the dependencies right after it is started, instead of waiting for the user to submit the first query ([#152](https://github.com/src-d/engine/issues/152)).
- All the `srcd parse drivers` management commands have been removed, except for `drivers list`. They are not needed anymore, now that source{d} Engine ships with pre-installed drivers, using fixed versions ([#85](https://github.com/src-d/engine/issues/85)).
- More user-friendly error message for unknown languages in the `srcd parse` output ([#163](https://github.com/src-d/engine/issues/163)).
