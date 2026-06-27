package service

import (
	"fmt"
	"strconv"
	"strings"
)

type metricOperatorFunc func(currentValue, targetValue string) (bool, error)

var metricOperatorEvaluators = map[string]metricOperatorFunc{
	"eq":     evalEq,
	"neq":    evalNeq,
	"gt":     numericOp(func(a, b float64) bool { return a > b }),
	"lt":     numericOp(func(a, b float64) bool { return a < b }),
	"ge":     numericOp(func(a, b float64) bool { return a >= b }),
	"le":     numericOp(func(a, b float64) bool { return a <= b }),
	"in":     evalIn,
	"not_in": evalNotIn,
}

func evaluateMetricOperator(operatorCode, currentValue, targetValue string) (bool, error) {
	eval, ok := metricOperatorEvaluators[operatorCode]
	if !ok {
		return false, fmt.Errorf("unsupported operator code %q", operatorCode)
	}
	return eval(currentValue, targetValue)
}

func evalEq(currentValue, targetValue string) (bool, error) {
	return currentValue == targetValue, nil
}

func evalNeq(currentValue, targetValue string) (bool, error) {
	return currentValue != targetValue, nil
}

func numericOp(cmp func(float64, float64) bool) metricOperatorFunc {
	return func(currentValue, targetValue string) (bool, error) {
		current, err := parseFloatValue(currentValue)
		if err != nil {
			return false, fmt.Errorf("parse current value %q: %w", currentValue, err)
		}
		target, err := parseFloatValue(targetValue)
		if err != nil {
			return false, fmt.Errorf("parse target value %q: %w", targetValue, err)
		}
		return cmp(current, target), nil
	}
}

func parseFloatValue(value string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(value), 64)
}

func parseTargetList(targetValue string) []string {
	parts := strings.Split(targetValue, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func evalIn(currentValue, targetValue string) (bool, error) {
	for _, item := range parseTargetList(targetValue) {
		if currentValue == item {
			return true, nil
		}
	}
	return false, nil
}

func evalNotIn(currentValue, targetValue string) (bool, error) {
	for _, item := range parseTargetList(targetValue) {
		if currentValue == item {
			return false, nil
		}
	}
	return true, nil
}
