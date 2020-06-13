package variables

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/jimlawless/whereami"
	kyverno "github.com/nirmata/kyverno/pkg/api/kyverno/v1"
	"github.com/nirmata/kyverno/pkg/engine/context"
	"github.com/nirmata/kyverno/pkg/engine/variables/operator"
)

//Evaluate evaluates the condition
func Evaluate(log logr.Logger, ctx context.EvalInterface, condition kyverno.Condition) bool {
	fmt.Printf("%s\n", whereami.WhereAmI())
	// get handler for the operator
	handle := operator.CreateOperatorHandler(log, ctx, condition.Operator, SubstituteVars)
	if handle == nil {
		return false
	}
	return handle.Evaluate(condition.Key, condition.Value)
}

//EvaluateConditions evaluates multiple conditions
func EvaluateConditions(log logr.Logger, ctx context.EvalInterface, conditions []kyverno.Condition) bool {
	fmt.Printf("%s\n", whereami.WhereAmI())
	// AND the conditions
	for _, condition := range conditions {
		if !Evaluate(log, ctx, condition) {
			return false
		}
	}
	return true
}
