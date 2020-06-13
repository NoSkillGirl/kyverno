package v1

import (
	"github.com/jimlawless/whereami"
	"reflect"
	"fmt"
)

//HasMutateOrValidateOrGenerate checks for rule types
func (p *ClusterPolicy) HasMutateOrValidateOrGenerate() bool {
	fmt.Printf("%s\n", whereami.WhereAmI())
	for _, rule := range p.Spec.Rules {
		if rule.HasMutate() || rule.HasValidate() || rule.HasGenerate() {
			return true
		}
	}
	return false
}

func (p *ClusterPolicy) BackgroundProcessingEnabled() bool {
	fmt.Printf("%s\n", whereami.WhereAmI())
	if p.Spec.Background == nil {
		return true
	}

	return *p.Spec.Background
}

//HasMutate checks for mutate rule
func (r Rule) HasMutate() bool {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return !reflect.DeepEqual(r.Mutation, Mutation{})
}

//HasValidate checks for validate rule
func (r Rule) HasValidate() bool {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return !reflect.DeepEqual(r.Validation, Validation{})
}

//HasGenerate checks for generate rule
func (r Rule) HasGenerate() bool {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return !reflect.DeepEqual(r.Generation, Generation{})
}

// DeepCopyInto is declared because k8s:deepcopy-gen is
// not able to generate this method for interface{} member
func (in *Mutation) DeepCopyInto(out *Mutation) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	if out != nil {
		*out = *in
	}
}

// DeepCopyInto is declared because k8s:deepcopy-gen is
// not able to generate this method for interface{} member
func (pp *Patch) DeepCopyInto(out *Patch) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	if out != nil {
		*out = *pp
	}
}

// DeepCopyInto is declared because k8s:deepcopy-gen is
// not able to generate this method for interface{} member
func (in *Validation) DeepCopyInto(out *Validation) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	if out != nil {
		*out = *in
	}
}

// DeepCopyInto is declared because k8s:deepcopy-gen is
// not able to generate this method for interface{} member
func (gen *Generation) DeepCopyInto(out *Generation) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	if out != nil {
		*out = *gen
	}
}

// DeepCopyInto is declared because k8s:deepcopy-gen is
// not able to generate this method for interface{} member
func (cond *Condition) DeepCopyInto(out *Condition) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	if out != nil {
		*out = *cond
	}
}

//ToKey generates the key string used for adding label to polivy violation
func (rs ResourceSpec) ToKey() string {
	return rs.Kind + "." + rs.Name
}
