package validate

import (
	"errors"
	"fmt"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/kyverno/kyverno/pkg/engine/anchor"
	commonAnchors "github.com/kyverno/kyverno/pkg/engine/anchor/common"
	"github.com/kyverno/kyverno/pkg/engine/common"
	"github.com/kyverno/kyverno/pkg/engine/operator"
	"github.com/kyverno/kyverno/pkg/engine/wildcards"
)

// ValidateResourceWithPattern is a start of element-by-element validation process
// It assumes that validation is started from root, so "/" is passed
func ValidateResourceWithPattern(log logr.Logger, resource, pattern interface{}) (string, error) {
	// newAnchorMap - to check anchor key has values
	ac := common.NewAnchorMap()
	elemPath, err := validateResourceElement(log, resource, pattern, pattern, "/", ac)
	if err != nil {
		if common.IsConditionalAnchorError(err.Error()) {
			return "", nil
		}

		if !ac.IsAnchorError() {
			return elemPath, err
		}
	}

	return "", nil
}

// validateResourceElement detects the element type (map, array, nil, string, int, bool, float)
// and calls corresponding handler
// Pattern tree and resource tree can have different structure. In this case validation fails
func validateResourceElement(log logr.Logger, resourceElement, patternElement, originPattern interface{}, path string, ac *common.AnchorKey) (string, error) {
	var err error
	switch typedPatternElement := patternElement.(type) {
	// map
	case map[string]interface{}:
		typedResourceElement, ok := resourceElement.(map[string]interface{})
		if !ok {
			log.V(4).Info("Pattern and resource have different structures.", "path", path, "expected", fmt.Sprintf("%T", patternElement), "current", fmt.Sprintf("%T", resourceElement))
			return path, fmt.Errorf("Pattern and resource have different structures. Path: %s. Expected %T, found %T", path, patternElement, resourceElement)
		}
		// CheckAnchorInResource - check anchor anchor key exists in resource and update the AnchorKey fields.
		ac.CheckAnchorInResource(typedPatternElement, typedResourceElement)
		return validateMap(log, typedResourceElement, typedPatternElement, originPattern, path, ac)
	// array
	case []interface{}:
		typedResourceElement, ok := resourceElement.([]interface{})
		if !ok {
			log.V(4).Info("Pattern and resource have different structures.", "path", path, "expected", fmt.Sprintf("%T", patternElement), "current", fmt.Sprintf("%T", resourceElement))
			return path, fmt.Errorf("Validation rule Failed at path %s, resource does not satisfy the expected overlay pattern", path)
		}
		return validateArray(log, typedResourceElement, typedPatternElement, originPattern, path, ac)
	// elementary values
	case string, float64, int, int64, bool, nil:
		/*Analyze pattern */
		if checkedPattern := reflect.ValueOf(patternElement); checkedPattern.Kind() == reflect.String {
			if isStringIsReference(checkedPattern.String()) { //check for $ anchor
				patternElement, err = actualizePattern(log, originPattern, checkedPattern.String(), path)
				if err != nil {
					return path, err
				}
			}
		}

		if !ValidateValueWithPattern(log, resourceElement, patternElement) {
			return path, fmt.Errorf("Validation rule failed at '%s' to validate value '%v' with pattern '%v'", path, resourceElement, patternElement)
		}

	default:
		log.V(4).Info("Pattern contains unknown type", "path", path, "current", fmt.Sprintf("%T", patternElement))
		return path, fmt.Errorf("Validation rule failed at '%s', pattern contains unknown type", path)
	}
	return "", nil
}

// func checkOrigPattern(origPattern interface{}) (string, error) {
// 	fmt.Println("\n---------checkOrigPattern-------")
// 	fmt.Println("origPattern: ", origPattern)
// 	switch typedOrigPattern := origPattern.(type) {
// 	case map[string]interface{}:
// 		for _, typedOrigPatternElement := range typedOrigPattern {
// 			checkOrigPattern(typedOrigPatternElement)
// 		}
// 	case []interface{}:
// 		for _, typedOrigPatternElement := range typedOrigPattern {
// 			checkOrigPattern(typedOrigPatternElement)
// 		}
// 	case string:
// 		return typedOrigPattern, nil
// 	default:
// 		fmt.Println("default-------type:- %T!!!!\n", origPattern)
// 	}
// 	return "", nil
// }

// func seperateOriginalPattern(origPattern map[string]interface{}, key string) interface{} {
// 	// var patterns map[string]interface{}
// 	patternKey := make([]string, 0)
// 	fmt.Println("\n---origPattern: ", origPattern)
// 	fmt.Println("key: ", key)
// 	for k, v := range origPattern {
// 		fmt.Println("k: ", k)
// 		fmt.Println("v: ", v)
// 		if k != key {
// 			typedV, ok := v.(map[string]interface{})
// 			if !ok {
// 				fmt.Println("1---typedV----type:- %T!!!!\n", typedV)
// 				fmt.Println("checking for array of interface........")
// 				typedV1, ok := v.([]interface{})
// 				if !ok {
// 					fmt.Println("2---typedV----type:- %T!!!!\n", typedV1)
// 				}
// 				for k1, v1 := range typedV1 {
// 					fmt.Println("k1: ", k1)
// 					fmt.Println("v1: ", v1)
// 				}
// 			}
// 			// fmt.Println("before----patterns:  ", patterns)
// 			// patterns[k] = seperateOriginalPattern(typedV, key)
// 			// fmt.Println("after ----patterns:  ", patterns)
// 			patternKey = append(patternKey, k)
// 			fmt.Println("patternKey:  ", patternKey)
// 		}
// 	}
// 	return "some stringggggggg"
// }

var patternKey []string

func getOriginalPatternKey(origPattern interface{}, key string) []string {
	fmt.Println("\n-------getOriginalPatternKey---------")
	fmt.Println("origPattern: ", origPattern)
	// patternKey := make([]string, 0)
	switch typedOrigPattern := origPattern.(type) {
	case map[string]interface{}:
		fmt.Println("===map[string]interface{}")
		for k, v := range typedOrigPattern {
			patternKey = append(patternKey, k)
			fmt.Println("patternKey: ", patternKey)
			getOriginalPatternKey(v, key)
		}
	case []interface{}:
		fmt.Println("===[]interface{}")
		for _, v := range typedOrigPattern {
			fmt.Println("v: ", v)
			combineMap(v)
		}
	}
	return nil
}

var combineOriginalPatternMap map[string]interface{}

func combineMap(v interface{}) {
	fmt.Println("#### v: ", v)
	l := len(patternKey) - 1
	for i := l; i >= 0; i-- {
		temp := make(map[string]interface{})
		fmt.Println("patternKey[i]: ", patternKey[i])
		if i == l {
			temp[patternKey[i]] = v
			combineOriginalPatternMap = temp
			fmt.Println("---temp: ", temp)
			fmt.Println("---combineOriginalPatternMap: ", combineOriginalPatternMap)
			continue
		}
		temp[patternKey[i]] = combineOriginalPatternMap
		combineOriginalPatternMap = temp
		fmt.Println("###temp: ", temp)
		fmt.Println("###combineOriginalPatternMap: ", combineOriginalPatternMap)
	}
	fmt.Println("############combineOriginalPatternMap: ", combineOriginalPatternMap)
}

// If validateResourceElement detects map element inside resource and pattern trees, it goes to validateMap
// For each element of the map we must detect the type again, so we pass these elements to validateResourceElement
func validateMap(log logr.Logger, resourceMap, patternMap map[string]interface{}, origPattern interface{}, path string, ac *common.AnchorKey) (string, error) {
	fmt.Println("\n---------validateMap-------")
	fmt.Println("resourceMap: ", resourceMap)
	fmt.Println("patternMap: ", patternMap)
	fmt.Println("origPattern: ", origPattern)

	// patternMap = wildcards.ExpandInMetadata(patternMap, resourceMap)
	// // check if there is anchor in pattern
	// // Phase 1 : Evaluate all the anchors
	// // Phase 2 : Evaluate non-anchors
	// anchors, resources := anchor.GetAnchorsResourcesFromMap(patternMap)

	// origPatternSting, err := checkOrigPattern(origPattern)
	// if err != nil {
	// 	fmt.Println("-----------error -----here")
	// }
	// fmt.Println("origPatternSting: ", origPatternSting)

	// // Evaluate anchors
	// for key, patternElement := range anchors {

	// 	// get handler for each pattern in the pattern
	// 	// - Conditional
	// 	// - Existence
	// 	// - Equality
	// 	handler := anchor.CreateElementHandler(key, patternElement, path)
	// 	handlerPath, err := handler.Handle(validateResourceElement, resourceMap, origPatternSting, ac)
	// 	// if there are resource values at same level, then anchor acts as conditional instead of a strict check
	// 	// but if there are non then its a if then check
	// 	if err != nil {
	// 		// If Conditional anchor fails then we don't process the resources
	// 		if commonAnchors.IsConditionAnchor(key) {
	// 			ac.AnchorError = common.NewConditionalAnchorError(fmt.Sprintf("condition anchor did not satisfy: %s", err.Error()))
	// 			log.V(3).Info(ac.AnchorError.Message)
	// 			return "", ac.AnchorError.Error()
	// 		}
	// 		return handlerPath, err
	// 	}
	// }

	// // Evaluate resources
	// // getSortedNestedAnchorResource - keeps the anchor key to start of the list
	// sortedResourceKeys := getSortedNestedAnchorResource(resources)
	// for e := sortedResourceKeys.Front(); e != nil; e = e.Next() {
	// 	key := e.Value.(string)
	// 	handler := anchor.CreateElementHandler(key, resources[key], path)
	// 	handlerPath, err := handler.Handle(validateResourceElement, resourceMap, origPatternSting, ac)
	// 	if err != nil {
	// 		return handlerPath, err
	// 	}
	// }

	// return "", nil

	// ----------------------------------------------------------------------------------
	patternMap = wildcards.ExpandInMetadata(patternMap, resourceMap)
	// check if there is anchor in pattern
	// Phase 1 : Evaluate all the anchors
	// Phase 2 : Evaluate non-anchors
	anchors, resources := anchor.GetAnchorsResourcesFromMap(patternMap)
	fmt.Println("anchors: ", anchors)
	// Evaluate anchors
	for key, patternElement := range anchors {
		if commonAnchors.IsExistenceAnchor(key) {
			fmt.Printf("type %T!\n", patternElement)
			typedPatternElement, ok := patternElement.([]interface{})
			if !ok {
				fmt.Println("type %T!\n", typedPatternElement)
			}
			for _, v := range typedPatternElement {
				fmt.Println("----typedPatternElement: ", v)
			}
			typedOrigPattern, ok := origPattern.(map[string]interface{})
			if !ok {
				fmt.Println("type %T!\n", typedOrigPattern)
			}
			a := getOriginalPatternKey(typedOrigPattern, key)
			fmt.Println("aaaaaaaa: ", a)
		}
		// get handler for each pattern in the pattern
		// - Conditional
		// - Existence
		// - Equality
		handler := anchor.CreateElementHandler(key, patternElement, path)
		fmt.Println("key: ", key)
		fmt.Println("patternElement: ", patternElement)
		fmt.Println("-----here---origPattern: ", origPattern)
		handlerPath, err := handler.Handle(validateResourceElement, resourceMap, origPattern, ac)
		// if there are resource values at same level, then anchor acts as conditional instead of a strict check
		// but if there are non then its a if then check
		if err != nil {
			// If Conditional anchor fails then we don't process the resources
			if commonAnchors.IsConditionAnchor(key) {
				ac.AnchorError = common.NewConditionalAnchorError(fmt.Sprintf("condition anchor did not satisfy: %s", err.Error()))
				log.V(3).Info(ac.AnchorError.Message)
				return "", ac.AnchorError.Error()
			}
			return handlerPath, err
		}
	}

	// Evaluate resources
	// getSortedNestedAnchorResource - keeps the anchor key to start of the list
	sortedResourceKeys := getSortedNestedAnchorResource(resources)
	for e := sortedResourceKeys.Front(); e != nil; e = e.Next() {
		key := e.Value.(string)
		handler := anchor.CreateElementHandler(key, resources[key], path)
		handlerPath, err := handler.Handle(validateResourceElement, resourceMap, origPattern, ac)
		if err != nil {
			return handlerPath, err
		}
	}
	return "", nil

}

func validateArray(log logr.Logger, resourceArray, patternArray []interface{}, originPattern interface{}, path string, ac *common.AnchorKey) (string, error) {

	if 0 == len(patternArray) {
		return path, fmt.Errorf("Pattern Array empty")
	}

	switch typedPatternElement := patternArray[0].(type) {
	case map[string]interface{}:
		// This is special case, because maps in arrays can have anchors that must be
		// processed with the special way affecting the entire array
		elemPath, err := validateArrayOfMaps(log, resourceArray, typedPatternElement, originPattern, path, ac)
		if err != nil {
			return elemPath, err
		}
	default:
		// In all other cases - detect type and handle each array element with validateResourceElement
		if len(resourceArray) >= len(patternArray) {
			for i, patternElement := range patternArray {
				currentPath := path + strconv.Itoa(i) + "/"
				elemPath, err := validateResourceElement(log, resourceArray[i], patternElement, originPattern, currentPath, ac)
				if err != nil {
					if common.IsConditionalAnchorError(err.Error()) {
						continue
					}
					return elemPath, err
				}
			}
		} else {
			return "", fmt.Errorf("Validate Array failed, array length mismatch, resource Array len is %d and pattern Array len is %d", len(resourceArray), len(patternArray))
		}
	}
	return "", nil
}

func actualizePattern(log logr.Logger, origPattern interface{}, referencePattern, absolutePath string) (interface{}, error) {
	var foundValue interface{}

	referencePattern = strings.Trim(referencePattern, "$()")

	operatorVariable := operator.GetOperatorFromStringPattern(referencePattern)
	referencePattern = referencePattern[len(operatorVariable):]

	if len(referencePattern) == 0 {
		return nil, errors.New("Expected path. Found empty reference")
	}
	// Check for variables
	// substitute it from Context
	// remove absolute path
	// {{ }}
	// value :=
	actualPath := formAbsolutePath(referencePattern, absolutePath)

	valFromReference, err := getValueFromReference(log, origPattern, actualPath)
	if err != nil {
		return err, nil
	}
	//TODO validate this
	if operatorVariable == operator.Equal { //if operator does not exist return raw value
		return valFromReference, nil
	}

	foundValue, err = valFromReferenceToString(valFromReference, string(operatorVariable))
	if err != nil {
		return "", err
	}
	return string(operatorVariable) + foundValue.(string), nil
}

//Parse value to string
func valFromReferenceToString(value interface{}, operator string) (string, error) {

	switch typed := value.(type) {
	case string:
		return typed, nil
	case int, int64:
		return fmt.Sprintf("%d", value), nil
	case float64:
		return fmt.Sprintf("%f", value), nil
	default:
		return "", fmt.Errorf("Incorrect expression. Operator %s does not match with value: %v", operator, value)
	}
}

// returns absolute path
func formAbsolutePath(referencePath, absolutePath string) string {
	if path.IsAbs(referencePath) {
		return referencePath
	}

	return path.Join(absolutePath, referencePath)
}

//Prepares original pattern, path to value, and call traverse function
func getValueFromReference(log logr.Logger, origPattern interface{}, reference string) (interface{}, error) {
	originalPatternMap := origPattern.(map[string]interface{})
	reference = reference[1:]
	statements := strings.Split(reference, "/")

	return getValueFromPattern(log, originalPatternMap, statements, 0)
}

func getValueFromPattern(log logr.Logger, patternMap map[string]interface{}, keys []string, currentKeyIndex int) (interface{}, error) {

	for key, pattern := range patternMap {
		rawKey := getRawKeyIfWrappedWithAttributes(key)

		if rawKey == keys[len(keys)-1] && currentKeyIndex == len(keys)-1 {
			return pattern, nil
		} else if rawKey != keys[currentKeyIndex] && currentKeyIndex != len(keys)-1 {
			continue
		}

		switch typedPattern := pattern.(type) {
		case []interface{}:
			if keys[currentKeyIndex] == rawKey {
				for i, value := range typedPattern {
					resourceMap, ok := value.(map[string]interface{})
					if !ok {
						log.V(4).Info("Pattern and resource have different structures.", "expected", fmt.Sprintf("%T", pattern), "current", fmt.Sprintf("%T", value))
						return nil, fmt.Errorf("Validation rule failed, resource does not have expected pattern %v", patternMap)
					}
					if keys[currentKeyIndex+1] == strconv.Itoa(i) {
						return getValueFromPattern(log, resourceMap, keys, currentKeyIndex+2)
					}
					// TODO : SA4004: the surrounding loop is unconditionally terminated (staticcheck)
					return nil, errors.New("Reference to non-existent place in the document")
				}
				return nil, nil // Just a hack to fix the lint
			}
			return nil, errors.New("Reference to non-existent place in the document")
		case map[string]interface{}:
			if keys[currentKeyIndex] == rawKey {
				return getValueFromPattern(log, typedPattern, keys, currentKeyIndex+1)
			}
			return nil, errors.New("Reference to non-existent place in the document")
		case string, float64, int, int64, bool, nil:
			continue
		}
	}

	elemPath := ""

	for _, elem := range keys {
		elemPath = "/" + elem + elemPath
	}
	return nil, fmt.Errorf("No value found for specified reference: %s", elemPath)
}

// validateArrayOfMaps gets anchors from pattern array map element, applies anchors logic
// and then validates each map due to the pattern
func validateArrayOfMaps(log logr.Logger, resourceMapArray []interface{}, patternMap map[string]interface{}, originPattern interface{}, path string, ac *common.AnchorKey) (string, error) {
	for i, resourceElement := range resourceMapArray {
		// check the types of resource element
		// expect it to be map, but can be anything ?:(
		currentPath := path + strconv.Itoa(i) + "/"
		returnpath, err := validateResourceElement(log, resourceElement, patternMap, originPattern, currentPath, ac)
		if err != nil {
			if common.IsConditionalAnchorError(err.Error()) {
				continue
			}
			return returnpath, err
		}
	}
	return "", nil
}

func isStringIsReference(str string) bool {
	if len(str) < len(operator.ReferenceSign) {
		return false
	}

	return str[0] == '$' && str[1] == '(' && str[len(str)-1] == ')'
}
