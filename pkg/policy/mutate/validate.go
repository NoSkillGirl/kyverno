package mutate

import (
	"errors"
	"fmt"

	"github.com/jimlawless/whereami"
	kyverno "github.com/nirmata/kyverno/pkg/api/kyverno/v1"
	"github.com/nirmata/kyverno/pkg/engine/anchor"
	"github.com/nirmata/kyverno/pkg/policy/common"
)

// Mutate provides implementation to validate 'mutate' rule
type Mutate struct {
	// rule to hold 'mutate' rule specifications
	rule kyverno.Mutation
}

//NewMutateFactory returns a new instance of Mutate validation checker
func NewMutateFactory(rule kyverno.Mutation) *Mutate {
	fmt.Printf("%s\n", whereami.WhereAmI())
	m := Mutate{
		rule: rule,
	}
	return &m
}

//Validate validates the 'mutate' rule
func (m *Mutate) Validate() (string, error) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	rule := m.rule
	// JSON Patches
	if len(rule.Patches) != 0 {
		for i, patch := range rule.Patches {
			if err := validatePatch(patch); err != nil {
				return fmt.Sprintf("patch[%d]", i), err
			}
		}
	}
	// Overlay
	if rule.Overlay != nil {
		path, err := common.ValidatePattern(rule.Overlay, "/", []anchor.IsAnchor{anchor.IsConditionAnchor, anchor.IsAddingAnchor})
		if err != nil {
			return path, err
		}
	}
	return "", nil
}

// Validate if all mandatory PolicyPatch fields are set
func validatePatch(pp kyverno.Patch) error {
	fmt.Printf("%s\n", whereami.WhereAmI())
	if pp.Path == "" {
		return errors.New("JSONPatch field 'path' is mandatory")
	}
	if pp.Operation == "add" || pp.Operation == "replace" {
		if pp.Value == nil {
			return fmt.Errorf("JSONPatch field 'value' is mandatory for operation '%s'", pp.Operation)
		}

		return nil
	} else if pp.Operation == "remove" {
		return nil
	}

	return fmt.Errorf("Unsupported JSONPatch operation '%s'", pp.Operation)
}
