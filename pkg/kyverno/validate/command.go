package validate

import (
	"fmt"
	"os"

	"github.com/jimlawless/whereami"
	"github.com/nirmata/kyverno/pkg/utils"

	"github.com/nirmata/kyverno/pkg/kyverno/common"
	"github.com/nirmata/kyverno/pkg/kyverno/sanitizedError"

	policyvalidate "github.com/nirmata/kyverno/pkg/policy"

	"github.com/spf13/cobra"
	log "sigs.k8s.io/controller-runtime/pkg/log"
)

func Command() *cobra.Command {
	fmt.Printf("%s\n", whereami.WhereAmI())
	cmd := &cobra.Command{
		Use:     "validate",
		Short:   "Validates kyverno policies",
		Example: "kyverno validate /path/to/policy.yaml /path/to/folderOfPolicies",
		RunE: func(cmd *cobra.Command, policyPaths []string) (err error) {
			defer func() {
				if err != nil {
					if !sanitizedError.IsErrorSanitized(err) {
						log.Log.Error(err, "failed to sanitize")
						err = fmt.Errorf("Internal error")
					}
				}
			}()

			policies, openAPIController, err := common.GetPoliciesValidation(policyPaths)
			if err != nil {
				return err
			}

			invalidPolicyFound := false
			for _, policy := range policies {
				err = policyvalidate.Validate(utils.MarshalPolicy(*policy), nil, true, openAPIController)
				if err != nil {
					fmt.Println("Policy " + policy.Name + " is invalid")
					invalidPolicyFound = true
				} else {
					fmt.Println("Policy " + policy.Name + " is valid")
				}
			}

			if invalidPolicyFound == true {
				os.Exit(1)
			}
			return nil
		},
	}
	return cmd
}
