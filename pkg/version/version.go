package version

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/jimlawless/whereami"
)

// These fields are set during an official build
// Global vars set from command-line arguments
var (
	BuildVersion = "--"
	BuildHash    = "--"
	BuildTime    = "--"
)

//PrintVersionInfo displays the kyverno version - git version
func PrintVersionInfo(log logr.Logger) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	log.Info("Kyverno", "Version", BuildVersion)
	log.Info("Kyverno", "BuildHash", BuildHash)
	log.Info("Kyverno", "BuildTime", BuildTime)
}
