package validate

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	v1 "github.com/kyverno/kyverno/pkg/api/kyverno/v1"
	"github.com/kyverno/kyverno/pkg/kyverno/common"
	sanitizederror "github.com/kyverno/kyverno/pkg/kyverno/sanitizedError"
	"github.com/kyverno/kyverno/pkg/openapi"
	policy2 "github.com/kyverno/kyverno/pkg/policy"
	"github.com/kyverno/kyverno/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/kube-openapi/pkg/util/proto/validation"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

// Command returns validate command
func Command() *cobra.Command {
	var outputType string
	var crdPaths []string
	cmd := &cobra.Command{
		Use:     "validate",
		Short:   "Validates kyverno policies",
		Example: "kyverno validate /path/to/policy.yaml /path/to/folderOfPolicies",
		RunE: func(cmd *cobra.Command, policyPaths []string) (err error) {
			defer func() {
				if err != nil {
					if !sanitizederror.IsErrorSanitized(err) {
						log.Log.Error(err, "failed to sanitize")
						err = fmt.Errorf("internal error")
					}
				}
			}()

			if outputType != "" {
				if outputType != "yaml" && outputType != "json" {
					return sanitizederror.NewWithError(fmt.Sprintf("%s format is not supported", outputType), errors.New("yaml and json are supported"))
				}
			}

			if len(policyPaths) == 0 {
				return sanitizederror.NewWithError(fmt.Sprintf("policy file(s) required"), err)
			}

			var policies []*v1.ClusterPolicy
			var errs []error
			if policyPaths[0] == "-" {
				if common.IsInputFromPipe() {
					policyStr := ""
					scanner := bufio.NewScanner(os.Stdin)
					for scanner.Scan() {
						policyStr = policyStr + scanner.Text() + "\n"
					}

					yamlBytes := []byte(policyStr)
					policies, err = utils.GetPolicy(yamlBytes)
					if err != nil {
						return sanitizederror.NewWithError("failed to parse policy", err)
					}
				}
			} else {
				policies, errs = common.GetPolicies(policyPaths)
				if len(errs) > 0 && len(policies) == 0 {
					return sanitizederror.NewWithErrors("failed to read policies", errs)
				}

				if len(errs) > 0 && log.Log.V(1).Enabled() {
					fmt.Printf("ignoring errors: \n")
					for _, e := range errs {
						fmt.Printf("    %v \n", e.Error())
					}
				}
			}

			openAPIController, err := openapi.NewOpenAPIController()
			if err != nil {
				return sanitizederror.NewWithError("failed to initialize openAPIController", err)
			}

			// if CRD's are passed, add these to OpenAPIController
			if len(crdPaths) > 0 {
				crds, err := common.GetCRDs(crdPaths)
				if err != nil {
					fmt.Printf("\nError: crd is invalid. \nFile: %s \nCause: %s\n", crdPaths, err)
					os.Exit(1)
				}
				for _, crd := range crds {
					openAPIController.ParseCRD(*crd)
				}
			}

			// convert these policies into policybytes
			// convert them to unstructured
			// validate according to crd
			kind := "ClusterPolicy"
			for _, policy := range policies {
				bytes, err := yaml.Marshal(policy)
				if err != nil {
					fmt.Println("error occured while converting policy into bytes")
					fmt.Println(err)
				}

				schema, err := openAPIController.GetCRDSchema(kind)
				if err != nil || schema == nil {
					fmt.Println("error occured while getting schema from openAPIController")
					fmt.Println(fmt.Errorf("pre-validation: couldn't find model %s, err: %v", kind, err))
				}

				unstructuredPolicy, err := common.ConvertResourceToUnstructured(bytes)
				if err != nil {
					fmt.Println("error occured while converting policy into unstructured")
					fmt.Println(err)
				}

				delete(unstructuredPolicy.Object, "kind")

				if errs := validation.ValidateModel(unstructuredPolicy.UnstructuredContent(), schema, kind); len(errs) > 0 {
					var errorMessages []string
					for i := range errs {
						errorMessages = append(errorMessages, errs[i].Error())
					}

					fmt.Println(fmt.Errorf(strings.Join(errorMessages, "\n\n")))
				}

			}

			invalidPolicyFound := false
			for _, policy := range policies {
				fmt.Println("----------------------------------------------------------------------")
				err := policy2.Validate(policy, nil, true, openAPIController)
				if err != nil {
					fmt.Printf("Policy %s is invalid.\n", policy.Name)
					fmt.Printf("Error: invalid policy.\nCause: %s\n\n", err)
					invalidPolicyFound = true
				} else {
					fmt.Printf("Policy %s is valid.\n\n", policy.Name)
					if outputType != "" {
						logger := log.Log.WithName("validate")
						p, err := common.MutatePolicy(policy, logger)
						if err != nil {
							if !sanitizederror.IsErrorSanitized(err) {
								return sanitizederror.NewWithError("failed to mutate policy.", err)
							}
							return err
						}
						if outputType == "yaml" {
							yamlPolicy, _ := yaml.Marshal(p)
							fmt.Println(string(yamlPolicy))
						} else {
							jsonPolicy, _ := json.MarshalIndent(p, "", "  ")
							fmt.Println(string(jsonPolicy))
						}
					}
				}
			}

			if invalidPolicyFound == true {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&outputType, "output", "o", "", "Prints the mutated policy in yaml or json format")
	cmd.Flags().StringArrayVarP(&crdPaths, "crd", "c", []string{}, "Path to CRD files")
	return cmd
}
