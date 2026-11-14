// Command ng-deploy backs up, builds, checks and copies a Niagara module's jars into a station modules
// directory. Run it through scripts/ng-deploy.sh, which builds this binary on demand and names itself in
// NG_DEPLOY_SCRIPT.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/ngdeploy"
)

func main() { cli.Run(ngdeploy.Main) }
