// Copyright © 2018 Francesc Campoy <francesc@sourced.tech>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/src-d/engine/cmd/srcd/cmd"
	"github.com/src-d/engine/cmd/srcd/daemon"
	"github.com/src-d/engine/components"
)

// this variable is rewritten during CI build step
var version = "dev"

func main() {
	cmd.SetVersion(version)
	daemon.SetCliVersion(version)
	components.SetCliVersion(version)
	cmd.Execute()
}
