package webhooks

import (
	contextdefault "context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kyverno "github.com/kyverno/kyverno/pkg/api/kyverno/v1"
	v1 "github.com/kyverno/kyverno/pkg/api/kyverno/v1"
	"github.com/kyverno/kyverno/pkg/config"
	"github.com/kyverno/kyverno/pkg/engine"
	"github.com/kyverno/kyverno/pkg/engine/context"
	"github.com/kyverno/kyverno/pkg/engine/response"
	enginutils "github.com/kyverno/kyverno/pkg/engine/utils"
	"github.com/kyverno/kyverno/pkg/engine/validate"
	"github.com/kyverno/kyverno/pkg/event"
	kyvernoutils "github.com/kyverno/kyverno/pkg/utils"
	"github.com/kyverno/kyverno/pkg/webhooks/generate"
	v1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
)

//HandleGenerate handles admission-requests for policies with generate rules
func (ws *WebhookServer) HandleGenerate(request *v1beta1.AdmissionRequest, policies []*kyverno.ClusterPolicy, ctx *context.Context, userRequestInfo kyverno.RequestInfo, dynamicConfig config.Interface) {
	logger := ws.log.WithValues("action", "generation", "uid", request.UID, "kind", request.Kind, "namespace", request.Namespace, "name", request.Name, "operation", request.Operation)
	logger.V(4).Info("incoming request")
	var engineResponses []*response.EngineResponse
	if request.Operation == v1beta1.Create || request.Operation == v1beta1.Update {
		if len(policies) == 0 {
			return
		}
		// convert RAW to unstructured
		new, old, err := kyvernoutils.ExtractResources(nil, request)
		if err != nil {
			logger.Error(err, "failed to extract resource")
		}

		policyContext := engine.PolicyContext{
			NewResource:         new,
			OldResource:         old,
			AdmissionInfo:       userRequestInfo,
			ExcludeGroupRole:    dynamicConfig.GetExcludeGroupRole(),
			ExcludeResourceFunc: ws.configHandler.ToFilter,
			ResourceCache:       ws.resCache,
			JSONContext:         ctx,
		}

		for _, policy := range policies {
			var rules []response.RuleResponse
			policyContext.Policy = *policy
			engineResponse := engine.Generate(policyContext)
			for _, rule := range engineResponse.PolicyResponse.Rules {
				if !rule.Success {
					ws.deleteGR(logger, engineResponse)
					continue
				}
				rules = append(rules, rule)
			}

			if len(rules) > 0 {
				engineResponse.PolicyResponse.Rules = rules
				// some generate rules do apply to the resource
				engineResponses = append(engineResponses, engineResponse)
				ws.statusListener.Update(generateStats{
					resp: engineResponse,
				})
			}
		}

		// Adds Generate Request to a channel(queue size 1000) to generators
		if failedResponse := applyGenerateRequest(ws.grGenerator, userRequestInfo, request.Operation, engineResponses...); err != nil {
			// report failure event
			for _, failedGR := range failedResponse {
				events := failedEvents(fmt.Errorf("failed to create Generate Request: %v", failedGR.err), failedGR.gr, new)
				ws.eventGen.Add(events...)
			}
		}
	}

	if request.Operation == v1beta1.Update {
		ws.handleUpdate(request, policies)
	}
}

//HandleUpdate handles admission-requests for update
func (ws *WebhookServer) handleUpdate(request *v1beta1.AdmissionRequest, policies []*kyverno.ClusterPolicy) {
	logger := ws.log.WithValues("action", "generation", "uid", request.UID, "kind", request.Kind, "namespace", request.Namespace, "name", request.Name, "operation", request.Operation)
	resource, err := enginutils.ConvertToUnstructured(request.OldObject.Raw)
	if err != nil {
		logger.Error(err, "failed to convert object resource to unstructured format")
	}

	resLabels := resource.GetLabels()
	if resLabels["generate.kyverno.io/clone-policy-name"] != "" {
		policyNames := strings.Split(resLabels["generate.kyverno.io/clone-policy-name"], ",")
		for _, policyName := range policyNames {
			selector := labels.SelectorFromSet(labels.Set(map[string]string{
				"generate.kyverno.io/policy-name": policyName,
			}))

			grList, err := ws.grLister.List(selector)
			if err != nil {
				logger.Error(err, "failed to get generate request for the resource", "label", "generate.kyverno.io/policy-name")

			}
			for _, gr := range grList {
				ws.grController.EnqueueGenerateRequestFromWebhook(gr)
			}
		}
	}

	enqueueBool := false
	if resLabels["app.kubernetes.io/managed-by"] == "kyverno" && resLabels["policy.kyverno.io/synchronize"] == "enable" && request.Operation == v1beta1.Update {
		// oldRes := resource
		newRes, err := enginutils.ConvertToUnstructured(request.Object.Raw)
		if err != nil {
			logger.Error(err, "failed to convert object resource to unstructured format")
		}
		// o, _ := json.Marshal(oldRes)
		// fmt.Println("\noldRes:      ", string(o))
		// n, _ := json.Marshal(newRes)
		// fmt.Println("\nnewRes:      ", string(n))

		policyName := resLabels["policy.kyverno.io/policy-name"]
		fmt.Println("policyNmae: ", policyName)
		targetSourceName := newRes.GetName()
		targetSourceKind := newRes.GetKind()

		for _, policy := range policies {
			if policy.GetName() == policyName {
				for _, rule := range policy.Spec.Rules {
					if rule.Generation.Kind == targetSourceKind && rule.Generation.Name == targetSourceName {
						data := rule.Generation.DeepCopy().Data
						if data != nil {
							fmt.Println("-----data is not nil-------")
							if path, err := validate.ValidateResourceWithPattern(logger, newRes.Object, data); err != nil {
								fmt.Println("path: ", path)
								enqueueBool = true
								break
							}
						}

						cloneName := rule.Generation.Clone.Name
						if cloneName != "" {
							fmt.Println("-----------generation with clone----------")
							obj, err := ws.client.GetResource("", rule.Generation.Kind, rule.Generation.Clone.Namespace, rule.Generation.Clone.Name)
							if err != nil {
								log.Log.Error(err, fmt.Sprintf("source resource %s/%s/%s not found.", rule.Generation.Kind, rule.Generation.Clone.Namespace, rule.Generation.Clone.Name))
								continue
							}
							o, _ := json.Marshal(obj)
							fmt.Println("\nclone source: ", string(o), "\n")

							unstructuredData, _, _ := unstructured.NestedMap(obj.Object, "data")
							unstructuredAnnotations := obj.GetAnnotations()

							delete(unstructuredAnnotations, "kubectl.kubernetes.io/last-applied-configuration")

							unstructuredAnnotationsInterface := make(map[string]interface{})
							for k, v := range unstructuredAnnotations {
								unstructuredAnnotationsInterface[k] = v
							}
							unstructuredLabels := obj.GetLabels()
							delete(unstructuredLabels, "generate.kyverno.io/clone-policy-name")

							unstructedMap := make(map[string]interface{})
							unstructuredMetaData := make(map[string]interface{})

							if len(unstructuredAnnotations) != 0 {
								unstructuredMetaData["annotations"] = unstructuredAnnotationsInterface

							}
							if len(unstructuredLabels) != 0 {
								unstructuredMetaData["labels"] = unstructuredLabels
							}

							unstructedMap["metadata"] = unstructuredMetaData

							unstructedMap["data"] = unstructuredData

							fmt.Println("unstructedMap: ", unstructedMap)
							if path, err := validate.ValidateResourceWithPattern(logger, newRes.Object, unstructedMap); err != nil {
								fmt.Println("path: ", path)
								// enqueueBool = true
								break
							} else {
								fmt.Println("passessssss......")
							}
						}
					}
				}
			}
		}

		if enqueueBool {
			grName := resLabels["policy.kyverno.io/gr-name"]
			gr, err := ws.grLister.Get(grName)
			if err != nil {
				logger.Error(err, "failed to get generate request", "name", grName)
			}
			fmt.Println("-------- enqueue ---------")
			ws.grController.EnqueueGenerateRequestFromWebhook(gr)
		}
	}
}

//HandleDelete handles admission-requests for delete
func (ws *WebhookServer) handleDelete(request *v1beta1.AdmissionRequest) {
	logger := ws.log.WithValues("action", "generation", "uid", request.UID, "kind", request.Kind, "namespace", request.Namespace, "name", request.Name, "operation", request.Operation)
	resource, err := enginutils.ConvertToUnstructured(request.OldObject.Raw)
	if err != nil {
		logger.Error(err, "failed to convert object resource to unstructured format")
	}

	resLabels := resource.GetLabels()
	if resLabels["app.kubernetes.io/managed-by"] == "kyverno" && resLabels["policy.kyverno.io/synchronize"] == "enable" && request.Operation == v1beta1.Delete {
		grName := resLabels["policy.kyverno.io/gr-name"]
		gr, err := ws.grLister.Get(grName)
		if err != nil {
			logger.Error(err, "failed to get generate request", "name", grName)
		}
		ws.grController.EnqueueGenerateRequestFromWebhook(gr)
	}
}

func (ws *WebhookServer) deleteGR(logger logr.Logger, engineResponse *response.EngineResponse) {
	logger.V(4).Info("querying all generate requests")
	selector := labels.SelectorFromSet(labels.Set(map[string]string{
		"generate.kyverno.io/policy-name":        engineResponse.PolicyResponse.Policy,
		"generate.kyverno.io/resource-name":      engineResponse.PolicyResponse.Resource.Name,
		"generate.kyverno.io/resource-kind":      engineResponse.PolicyResponse.Resource.Kind,
		"generate.kyverno.io/resource-namespace": engineResponse.PolicyResponse.Resource.Namespace,
	}))

	grList, err := ws.grLister.List(selector)
	if err != nil {
		logger.Error(err, "failed to get generate request for the resource", "kind", engineResponse.PolicyResponse.Resource.Kind, "name", engineResponse.PolicyResponse.Resource.Name, "namespace", engineResponse.PolicyResponse.Resource.Namespace)

	}

	for _, v := range grList {
		err := ws.kyvernoClient.KyvernoV1().GenerateRequests(config.KyvernoNamespace).Delete(contextdefault.TODO(), v.GetName(), metav1.DeleteOptions{})
		if err != nil {
			logger.Error(err, "failed to update gr")
		}
	}
}

func applyGenerateRequest(gnGenerator generate.GenerateRequests, userRequestInfo kyverno.RequestInfo,
	action v1beta1.Operation, engineResponses ...*response.EngineResponse) (failedGenerateRequest []generateRequestResponse) {

	for _, er := range engineResponses {
		gr := transform(userRequestInfo, er)
		if err := gnGenerator.Apply(gr, action); err != nil {
			failedGenerateRequest = append(failedGenerateRequest, generateRequestResponse{gr: gr, err: err})
		}
	}

	return
}

func transform(userRequestInfo kyverno.RequestInfo, er *response.EngineResponse) kyverno.GenerateRequestSpec {
	gr := kyverno.GenerateRequestSpec{
		Policy: er.PolicyResponse.Policy,
		Resource: kyverno.ResourceSpec{
			Kind:      er.PolicyResponse.Resource.Kind,
			Namespace: er.PolicyResponse.Resource.Namespace,
			Name:      er.PolicyResponse.Resource.Name,
		},
		Context: kyverno.GenerateRequestContext{
			UserRequestInfo: userRequestInfo,
		},
	}

	return gr
}

type generateStats struct {
	resp *response.EngineResponse
}

func (gs generateStats) PolicyName() string {
	return gs.resp.PolicyResponse.Policy
}

func (gs generateStats) UpdateStatus(status kyverno.PolicyStatus) kyverno.PolicyStatus {
	if reflect.DeepEqual(response.EngineResponse{}, gs.resp) {
		return status
	}

	var nameToRule = make(map[string]v1.RuleStats)
	for _, rule := range status.Rules {
		nameToRule[rule.Name] = rule
	}

	for _, rule := range gs.resp.PolicyResponse.Rules {
		ruleStat := nameToRule[rule.Name]
		ruleStat.Name = rule.Name

		averageOver := int64(ruleStat.AppliedCount + ruleStat.FailedCount)
		ruleStat.ExecutionTime = updateAverageTime(
			rule.ProcessingTime,
			ruleStat.ExecutionTime,
			averageOver).String()

		if rule.Success {
			status.RulesAppliedCount++
			ruleStat.AppliedCount++
		} else {
			status.RulesFailedCount++
			ruleStat.FailedCount++
		}

		nameToRule[rule.Name] = ruleStat
	}

	var policyAverageExecutionTime time.Duration
	var ruleStats = make([]v1.RuleStats, 0, len(nameToRule))
	for _, ruleStat := range nameToRule {
		executionTime, err := time.ParseDuration(ruleStat.ExecutionTime)
		if err == nil {
			policyAverageExecutionTime += executionTime
		}
		ruleStats = append(ruleStats, ruleStat)
	}

	sort.Slice(ruleStats, func(i, j int) bool {
		return ruleStats[i].Name < ruleStats[j].Name
	})

	status.AvgExecutionTime = policyAverageExecutionTime.String()
	status.Rules = ruleStats

	return status
}

func updateAverageTime(newTime time.Duration, oldAverageTimeString string, averageOver int64) time.Duration {
	if averageOver == 0 {
		return newTime
	}
	oldAverageExecutionTime, _ := time.ParseDuration(oldAverageTimeString)
	numerator := (oldAverageExecutionTime.Nanoseconds() * averageOver) + newTime.Nanoseconds()
	denominator := averageOver + 1
	newAverageTimeInNanoSeconds := numerator / denominator
	return time.Duration(newAverageTimeInNanoSeconds) * time.Nanosecond
}

type generateRequestResponse struct {
	gr  v1.GenerateRequestSpec
	err error
}

func (resp generateRequestResponse) info() string {
	return strings.Join([]string{resp.gr.Resource.Kind, resp.gr.Resource.Namespace, resp.gr.Resource.Name}, "/")
}

func (resp generateRequestResponse) error() string {
	return resp.err.Error()
}

func failedEvents(err error, gr kyverno.GenerateRequestSpec, resource unstructured.Unstructured) []event.Info {
	re := event.Info{}
	re.Kind = resource.GetKind()
	re.Namespace = resource.GetNamespace()
	re.Name = resource.GetName()
	re.Reason = event.PolicyFailed.String()
	re.Source = event.GeneratePolicyController
	re.Message = fmt.Sprintf("policy %s failed to apply: %v", gr.Policy, err)

	return []event.Info{re}
}
