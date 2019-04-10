# regression-core

*regression-core** holds the common functionality used by regression testers.

Functionality provided:

* Binary download from github releases
* Binary building from local or remote locations, can specify tag/branch or pull request
* Command execution and resource retrieval (max memory, wall/system/user time)
* `git` server execution
* Generic server execution
* Repository cache management

The library should be imported using `gopkg.in`:

```go
import (
  "gopkg.in/src-d/regression-core.v0
)

...

config := regression.NewConfig()
config.RepositoriesCache = "/tmp/repos"
git := regression.NewGitServer(config)
git.Start()
```

## License

Licensed under the terms of the Apache License Version 2.0. See the `LICENSE`
file for the full license text.
