package collector

import (
	"math"

	"github.com/Knetic/govaluate"
)

var functions map[string]govaluate.ExpressionFunction

func init() {
	defer trace()()
	functions = map[string]govaluate.ExpressionFunction{
		"sum":     sum,
		"average": average,
		"count":   count,
		"max":     max,
		"min":     min,
	}
}

func sum(args ...interface{}) (interface{}, error) {
	defer trace()()
	s := 0.
	for _, x := range args {
		switch x.(type) {
		case []interface{}:
			y, _ := sum(x)
			s += y.(float64)
		default:
			s += x.(float64)
		}
	}
	return s, nil
}

func average(args ...interface{}) (interface{}, error) {
	defer trace()()
	s, _ := sum(args...)
	c, _ := count(args...)

	return (s.(float64) / c.(float64)), nil
}

func count(args ...interface{}) (interface{}, error) {
	defer trace()()
	s := len(args)
	for _, x := range args {
		switch x.(type) {
		case []interface{}:
			y, _ := count(x)
			s += y.(int)
		}
	}
	return float64(s), nil
}

func max(args ...interface{}) (interface{}, error) {
	defer trace()()
	s := 0.
	for _, x := range args {
		switch x.(type) {
		case []interface{}:
			y, _ := max(x)
			s = math.Max(y.(float64), s)
		default:
			s = math.Max(x.(float64), s)
		}
	}
	return s, nil
}

func min(args ...interface{}) (interface{}, error) {
	defer trace()()
	s := 0.
	for _, x := range args {
		switch x.(type) {
		case []interface{}:
			y, _ := min(x)
			s = math.Min(y.(float64), s)
		default:
			s = math.Min(x.(float64), s)
		}
	}
	return s, nil
}
