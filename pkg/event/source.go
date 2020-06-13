package event

import (
	"fmt"

	"github.com/jimlawless/whereami"
)

//Source of event generation
type Source int

const (
	// AdmissionController : event generated in admission-control webhook
	AdmissionController Source = iota
	// PolicyController : event generated in policy-controller
	PolicyController
	// GeneratePolicyController : event generated in generate policyController
	GeneratePolicyController
)

func (s Source) String() string {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return [...]string{
		"admission-controller",
		"policy-controller",
		"generate-policy-controller",
	}[s]
}
