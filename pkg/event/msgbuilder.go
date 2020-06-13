package event

import (
	"fmt"
	"regexp"

	"github.com/jimlawless/whereami"
)

//MsgKey is an identified to determine the preset message formats
type MsgKey int

//Message id for pre-defined messages
const (
	FResourcePolcy MsgKey = iota
	FProcessRule
	SPolicyApply
	SRulesApply
	FPolicyApplyBlockCreate
	FPolicyApplyBlockUpdate
	FPolicyBlockResourceUpdate
	FPolicyApplyFailed
	FResourcePolicyFailed
)

func (k MsgKey) String() string {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return [...]string{
		"Policy violation on resource '%s'. The rule(s) '%s' not satisfied",
		"Failed to process rule '%s' of policy '%s'.",
		"Policy applied successfully on the resource '%s'",
		"Rule(s) '%s' of Policy '%s' applied successfully",
		"Resource %s creation blocked by rule(s) %s",
		"Rule(s) '%s' of policy '%s' blocked update of the resource",
		"Resource %s update blocked by rule(s) %s",
		"Rule(s) '%s' failed to apply on resource %s",
		"Rule(s) '%s' of policy '%s' failed to apply on the resource",
	}[k]
}

const argRegex = "%[s,d,v]"

var re = regexp.MustCompile(argRegex)

//GetEventMsg return the application message based on the message id and the arguments,
// if the number of arguments passed to the message are incorrect generate an error
func getEventMsg(key MsgKey, args ...interface{}) (string, error) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	// Verify the number of arguments
	argsCount := len(re.FindAllString(key.String(), -1))
	if argsCount != len(args) {
		return "", fmt.Errorf("message expects %d arguments, but %d arguments passed", argsCount, len(args))
	}
	return fmt.Sprintf(key.String(), args...), nil
}
