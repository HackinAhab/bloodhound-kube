package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ExpressionEvaluator evaluates simple expressions for computed properties
type ExpressionEvaluator struct {
	functions map[string]ExpressionFunc
}

// ExpressionFunc is a function that can be called in expressions
type ExpressionFunc func(args []any) (any, error)

// NewExpressionEvaluator creates a new expression evaluator
func NewExpressionEvaluator() *ExpressionEvaluator {
	eval := &ExpressionEvaluator{
		functions: make(map[string]ExpressionFunc),
	}

	// Register built-in functions
	eval.registerBuiltinFunctions()

	return eval
}

// Evaluate evaluates an expression against a resource context
func (ee *ExpressionEvaluator) Evaluate(expression string, context map[string]any) (any, error) {
	// Trim whitespace
	expression = strings.TrimSpace(expression)

	// Parse and evaluate the expression
	return ee.parseAndEvaluate(expression, context)
}

// parseAndEvaluate parses and evaluates a simple expression
func (ee *ExpressionEvaluator) parseAndEvaluate(expr string, context map[string]any) (any, error) {
	// Handle function calls: function_name(arg1, arg2, ...)
	if strings.Contains(expr, "(") {
		return ee.evaluateFunctionCall(expr, context)
	}

	// Handle simple variable reference
	if val, exists := context[expr]; exists {
		return val, nil
	}

	// Handle string literals
	if strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") {
		return strings.Trim(expr, "'"), nil
	}
	if strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") {
		return strings.Trim(expr, "\""), nil
	}

	// Handle numeric literals
	if num, err := strconv.Atoi(expr); err == nil {
		return num, nil
	}
	if num, err := strconv.ParseFloat(expr, 64); err == nil {
		return num, nil
	}

	// Handle boolean literals
	if expr == "true" {
		return true, nil
	}
	if expr == "false" {
		return false, nil
	}

	return nil, fmt.Errorf("unable to evaluate expression: %s", expr)
}

// evaluateFunctionCall evaluates a function call expression
func (ee *ExpressionEvaluator) evaluateFunctionCall(expr string, context map[string]any) (any, error) {
	// Find function name and arguments
	parenIndex := strings.Index(expr, "(")
	if parenIndex == -1 {
		return nil, fmt.Errorf("invalid function call syntax: %s", expr)
	}

	functionName := strings.TrimSpace(expr[:parenIndex])
	argsStr := strings.TrimSpace(expr[parenIndex+1:])

	// Remove closing parenthesis
	if !strings.HasSuffix(argsStr, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in: %s", expr)
	}
	argsStr = strings.TrimSuffix(argsStr, ")")

	// Parse arguments
	args, err := ee.parseArguments(argsStr, context)
	if err != nil {
		return nil, fmt.Errorf("failed to parse arguments in %s: %w", expr, err)
	}

	// Look up and call the function
	fn, exists := ee.functions[functionName]
	if !exists {
		return nil, fmt.Errorf("unknown function: %s", functionName)
	}

	return fn(args)
}

// parseArguments parses comma-separated function arguments
func (ee *ExpressionEvaluator) parseArguments(argsStr string, context map[string]any) ([]any, error) {
	if strings.TrimSpace(argsStr) == "" {
		return []any{}, nil
	}

	var args []any
	var currentArg strings.Builder
	inQuotes := false
	inBrackets := 0
	quoteChar := rune(0)

	for _, ch := range argsStr {
		switch {
		case (ch == '\'' || ch == '"') && quoteChar == 0:
			quoteChar = ch
			inQuotes = true
			currentArg.WriteRune(ch)
		case ch == quoteChar:
			quoteChar = 0
			inQuotes = false
			currentArg.WriteRune(ch)
		case ch == '[' && !inQuotes:
			inBrackets++
			currentArg.WriteRune(ch)
		case ch == ']' && !inQuotes:
			inBrackets--
			currentArg.WriteRune(ch)
		case ch == ',' && !inQuotes && inBrackets == 0:
			// Argument separator
			arg, err := ee.parseAndEvaluate(strings.TrimSpace(currentArg.String()), context)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			currentArg.Reset()
		default:
			currentArg.WriteRune(ch)
		}
	}

	// Add last argument
	if currentArg.Len() > 0 {
		arg, err := ee.parseAndEvaluate(strings.TrimSpace(currentArg.String()), context)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	return args, nil
}

// RegisterFunction registers a custom expression function
func (ee *ExpressionEvaluator) RegisterFunction(name string, fn ExpressionFunc) {
	ee.functions[name] = fn
}

// registerBuiltinFunctions registers the built-in expression functions
func (ee *ExpressionEvaluator) registerBuiltinFunctions() {
	// len: Get length of array, map, or string
	ee.RegisterFunction("len", func(args []any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("len() requires exactly 1 argument, got %d", len(args))
		}

		v := reflect.ValueOf(args[0])
		switch v.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
			return v.Len(), nil
		default:
			return 0, nil
		}
	})

	// contains: Check if a collection contains a value
	ee.RegisterFunction("contains", func(args []any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("contains() requires exactly 2 arguments, got %d", len(args))
		}

		collection := args[0]
		searchValue := args[1]

		// Handle slice
		if slice, ok := collection.([]any); ok {
			for _, item := range slice {
				if reflect.DeepEqual(item, searchValue) {
					return true, nil
				}
			}
			return false, nil
		}

		// Handle string slice
		if slice, ok := collection.([]string); ok {
			searchStr, ok := searchValue.(string)
			if !ok {
				return false, nil
			}
			for _, item := range slice {
				if item == searchStr {
					return true, nil
				}
			}
			return false, nil
		}

		// Handle map (check for key)
		if m, ok := collection.(map[string]any); ok {
			key, ok := searchValue.(string)
			if !ok {
				return false, nil
			}
			_, exists := m[key]
			return exists, nil
		}

		return false, nil
	})

	// contains_any: Check if collection contains any of the specified values
	ee.RegisterFunction("contains_any", func(args []any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("contains_any() requires exactly 2 arguments, got %d", len(args))
		}

		collection := args[0]
		searchValues := args[1]

		// Convert searchValues to slice
		searchSlice, ok := searchValues.([]any)
		if !ok {
			return false, fmt.Errorf("contains_any() second argument must be an array")
		}

		// Convert collection to string slice if possible
		var collectionStrs []string
		if slice, ok := collection.([]string); ok {
			collectionStrs = slice
		} else if slice, ok := collection.([]any); ok {
			for _, item := range slice {
				if str, ok := item.(string); ok {
					collectionStrs = append(collectionStrs, str)
				}
			}
		} else {
			return false, nil
		}

		// Check if any search value is in collection
		for _, searchVal := range searchSlice {
			searchStr, ok := searchVal.(string)
			if !ok {
				continue
			}
			for _, collStr := range collectionStrs {
				if strings.Contains(strings.ToLower(collStr), strings.ToLower(searchStr)) {
					return true, nil
				}
			}
		}

		return false, nil
	})

	// any: Check if any item in array satisfies a condition (simplified version)
	ee.RegisterFunction("any", func(args []any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("any() requires exactly 1 argument, got %d", len(args))
		}

		// Convert to slice of booleans
		if slice, ok := args[0].([]any); ok {
			for _, item := range slice {
				if boolVal, ok := item.(bool); ok && boolVal {
					return true, nil
				}
			}
		}

		return false, nil
	})

	// all: Check if all items in array satisfy a condition (simplified version)
	ee.RegisterFunction("all", func(args []any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("all() requires exactly 1 argument, got %d", len(args))
		}

		// Convert to slice of booleans
		if slice, ok := args[0].([]any); ok {
			for _, item := range slice {
				if boolVal, ok := item.(bool); ok && !boolVal {
					return false, nil
				}
			}
			return true, nil
		}

		return true, nil
	})
}
