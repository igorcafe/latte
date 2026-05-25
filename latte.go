package latte

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"unicode"
)

type Env struct {
	functions map[symbol]func([]any) any
	variables map[symbol]any
	parent    *Env
}

func NewEnv() *Env {
	return &Env{
		functions: maps.Clone(stdlibFunctions),
		variables: maps.Clone(stdlibVariables),
		parent:    nil,
	}
}

func (env *Env) Eval(source string) (value any, err error) {
	defer func() {
		if r := recover(); r != nil {
			if errValue, ok := r.(error); ok {
				err = errValue
				return
			}
			err = fmt.Errorf("%v", r)
		}
	}()

	var result any
	for _, ast := range parseProgram(tokenize(source)) {
		result = eval(ast, env)
	}
	return result, nil
}

func (env *Env) Define(name string, value any) {
	env.defineVariable(symbol(name), value)
}

func (env *Env) RegisterFunction(name string, fn func([]any) any) {
	env.functions[symbol(name)] = fn
}

var stdlibVariables = map[symbol]any{
	"true":  true,
	"false": false,
	"nil":   nil,
}

var stdlibFunctions = map[symbol]func(args []any) any{
	"+": func(args []any) any {
		return arithmeticOperation(
			args,
			func(a, b float64) float64 { return a + b },
		)
	},
	"-": func(args []any) any {
		return arithmeticOperation(
			args,
			func(a, b float64) float64 { return a - b },
		)
	},
	"*": func(args []any) any {
		return arithmeticOperation(
			args,
			func(a, b float64) float64 { return a * b },
		)
	},
	"/": func(args []any) any {
		return arithmeticOperation(
			args,
			func(a, b float64) float64 { return a / b },
		)
	},
	"not": func(args []any) any {
		if len(args) != 1 {
			panic("TODO wrong number of args (not)")
		}
		return !isTruthy(args[0])
	},
	"=": lispEqual,
	"!=": func(args []any) any {
		return !lispEqual(args).(bool)
	},
	">": func(args []any) any {
		return lispCompare(args, ">")
	},
	">=": func(args []any) any {
		return lispCompare(args, ">=")
	},
	"<": func(args []any) any {
		return lispCompare(args, "<")
	},
	"<=": func(args []any) any {
		return lispCompare(args, "<=")
	},
	"list": func(args []any) any {
		return args
	},
	"car": func(args []any) any {
		if len(args) != 1 {
			panic("car expects exactly one argument")
		}

		if args[0] == nil || isEmptyList(args[0]) {
			return nil
		}

		list, ok := args[0].([]any)
		if !ok {
			panic("car expects a list")
		}

		return list[0]
	},
	"cdr": func(args []any) any {
		if len(args) != 1 {
			panic("cdr expects exactly one argument")
		}

		if args[0] == nil || isEmptyList(args[0]) {
			return nil
		}

		list, ok := args[0].([]any)
		if !ok {
			panic("cdr expects a list")
		}

		if len(list) <= 1 {
			return nil
		}

		return list[1:]
	},
	"nth": func(args []any) any {
		if len(args) != 2 {
			panic("nth expects exactly two arguments")
		}

		index, ok := args[0].(float64)
		if !ok {
			panic("nth expects index to be a number")
		}
		if index < 0 || index != float64(int(index)) {
			panic("nth expects index to be a non-negative integer")
		}

		if args[1] == nil || isEmptyList(args[1]) {
			return nil
		}

		list, ok := args[1].([]any)
		if !ok {
			panic("nth expects a list")
		}

		i := int(index)
		if i >= len(list) {
			return nil
		}

		return list[i]
	},
	"null": func(args []any) any {
		if len(args) != 1 {
			panic("null expects exactly one argument")
		}

		return args[0] == nil || isEmptyList(args[0])
	},
	"length": func(args []any) any {
		if len(args) != 1 {
			panic("length expects exactly one argument")
		}

		if args[0] == nil {
			return 0.0
		}

		list, ok := args[0].([]any)
		if !ok {
			panic("length expects a list")
		}

		return float64(len(list))
	},
	"print": func(args []any) any {
		fmt.Print(args...)
		return nil
	},
	"println": func(args []any) any {
		fmt.Println(args...)
		return nil
	},
	"printf": func(args []any) any {
		if len(args) == 0 {
			panic("printf: invalid number of arguments")
		}
		if _, ok := args[0].(string); !ok {
			panic("printf: invalid argument type for format string")
		}

		fmt.Printf(args[0].(string), args[1:]...)
		return nil
	},
}

func lispEqual(args []any) any {
	if len(args) == 0 {
		return true
	}
	val := args[0]
	for _, arg := range args {
		if !reflect.DeepEqual(val, arg) {
			return false
		}
	}
	return true
}

func lispCompare(args []any, op string) any {
	if len(args) < 2 {
		panic(op + ": expected at least 2 arguments")
	}

	for i := range args[:len(args)-1] {
		a := args[i]
		b := args[i+1]
		if reflect.TypeOf(a) != reflect.TypeOf(b) {
			panic("comparison of different types")
		}
		switch a := a.(type) {
		case string:
			if !compareOrdered(a, b.(string), op) {
				return false
			}
		case float64:
			if !compareOrdered(a, b.(float64), op) {
				return false
			}
		default:
			panic("type is not comparable")
		}
	}
	return true
}

func compareOrdered[T cmp.Ordered](a, b T, op string) bool {
	switch op {
	case ">":
		return a > b
	case ">=":
		return a >= b
	case "<":
		return a < b
	case "<=":
		return a <= b
	default:
		panic("unknown comparison operator: " + op)
	}
}

func arithmeticOperation(args []any, op func(float64, float64) float64) float64 {
	if len(args) == 0 {
		return 0.0
	}

	total := 0.0
	for i, arg := range args {
		switch arg := arg.(type) {
		case float64:
			if i == 0 {
				total = arg
			} else {
				total = op(total, arg)
			}
		default:
			panic("TODO: user level error for non float")
		}
	}

	return total
}

type symbol string

func tokenize(source string) []string {
	tokens := []string{}
	source = strings.TrimSpace(source)
	current := ""
	flushCurrent := func() {
		if current != "" {
			tokens = append(tokens, current)
			current = ""
		}
	}

	inString := false
	prevRune := rune(0)

	parens := 0

	for _, r := range source {
		if r == '"' && prevRune != '\\' {
			inString = !inString
		}
		prevRune = r
		if !inString && unicode.IsSpace(r) {
			flushCurrent()
			continue
		}
		if !inString && r == '\'' {
			flushCurrent()
			tokens = append(tokens, string(r))
			continue
		}
		if !inString && slices.Contains([]rune{'(', ')'}, r) {
			if r == '(' {
				parens++
			} else {
				parens--
			}
			flushCurrent()
			tokens = append(tokens, string(r))
			continue
		}
		current += string(r)
	}

	if current != "" {
		tokens = append(tokens, current)
	}

	if parens != 0 {
		panic("unbalanced parenthesis")
	}

	return tokens
}

func parse(tokens []string) any {
	if len(tokens) == 0 {
		return nil
	}

	pos := 0
	expr := parseExpr(tokens, &pos)
	if pos != len(tokens) {
		panic("TODO: syntax error, unexpected tokens")
	}

	return expr

}

func parseProgram(tokens []string) []any {
	exprs := []any{}
	pos := 0
	for pos < len(tokens) {
		exprs = append(exprs, parseExpr(tokens, &pos))
	}
	return exprs
}

func parseExpr(tokens []string, pos *int) any {
	if *pos >= len(tokens) {
		panic("TODO: syntax error, unexpected end of input")
	}

	token := tokens[*pos]
	*pos++

	switch token {
	case "'":
		return []any{symbol("quote"), parseExpr(tokens, pos)}
	case "(":
		tree := []any{}
		for {
			if *pos >= len(tokens) {
				panic("TODO: syntax error, unbalanced parenthesis")
			}
			if tokens[*pos] == ")" {
				*pos++
				return tree
			}
			tree = append(tree, parseExpr(tokens, pos))
		}
	case ")":
		panic("TODO: syntax error, unexpected closing parenthesis")
	default:
		return parseAtom(token)
	}
}

func parseAtom(token string) any {
	var value any
	if err := json.Unmarshal([]byte(token), &value); err == nil {
		switch value.(type) {
		case string, int, float64:
			// avoid converting "true", "null" and things like that
			return value
		}
	}
	return symbol(token)
}

func (env *Env) findSymbol(symbol symbol) (any, bool) {
	if variable, ok := env.variables[symbol]; ok {
		return variable, ok
	}
	if env.parent != nil {
		return env.parent.findSymbol(symbol)
	}
	return nil, false
}

func (env *Env) defineVariable(name symbol, value any) {
	env.variables[name] = value
}

func (env *Env) childScope() *Env {
	return &Env{
		functions: env.functions,
		variables: make(map[symbol]any),
		parent:    env,
	}
}

func eval(node any, env *Env) any {
	switch node := node.(type) {
	case symbol:
		value, ok := env.findSymbol(node)
		if !ok {
			panic("unknown symbol: " + string(node))
		}
		return value
	case string, int, float64:
		return node
	case []any:
		if len(node) == 0 {
			return nil
		}

		operator := node[0].(symbol)

		// special functions
		switch operator {
		case "let", "let*":
			if len(node) == 1 {
				return nil
			}
			bindings, ok := node[1].([]any)
			if !ok {
				panic("let expression expected bindings at position 2")
			}

			scope := env.childScope()
			for _, binding := range bindings {
				binding, ok := binding.([]any)
				if !ok || len(binding) != 2 {
					panic("let expression invalid binding")
				}
				name, ok := binding[0].(symbol)
				if !ok {
					panic("let binding is not a symbol: " + fmt.Sprint(binding[0]))
				}

				if string(operator) == "let" {
					scope.defineVariable(name, eval(binding[1], env))
				} else {
					scope.defineVariable(name, eval(binding[1], scope))
				}
			}

			if len(node) == 2 {
				return nil
			}

			var lastEval any
			for _, node := range node[2:] {
				lastEval = eval(node, scope)
			}
			return lastEval

		case "define":
			if len(node) != 3 {
				panic("define expects exactly 2 arguments")
			}
			name, ok := node[1].(symbol)
			if !ok {
				panic("define argument 1 must be a symbol")
			}
			value := eval(node[2], env)
			env.defineVariable(name, value)
			return value

		case "when", "unless":
			if len(node) < 2 {
				panic(string(operator) + " needs a condition")
			}

			cond := eval(node[1], env)
			if isTruthy(cond) == (string(operator) == "unless") {
				return nil
			}

			var lastEval any
			for _, node := range node[2:] {
				lastEval = eval(node, env)
			}

			return lastEval

		case "if":
			if len(node) < 2 {
				panic("if needs a condition")
			}

			cond := eval(node[1], env)
			if len(node) == 2 {
				return nil
			}

			if isTruthy(cond) {
				return eval(node[2], env)
			}

			var lastEval any
			for _, node := range node[3:] {
				lastEval = eval(node, env)
			}

			return lastEval

		case "progn":
			var lastEval any
			for _, node := range node[1:] {
				lastEval = eval(node, env)
			}

			return lastEval

		case "quote":
			if len(node) != 2 {
				panic("quote expects exactly one argument")
			}
			return cloneValue(node[1])
		}

		function, ok := env.functions[operator]
		if !ok {
			panic("unknown function " + string(operator))
		}

		args := []any{}
		if len(node) > 1 {
			for _, node := range node[1:] {
				arg := eval(node, env)
				args = append(args, arg)
			}
		}

		result := function(args)
		return result
	}

	return nil
}
func isTruthy(val any) bool {
	if val == nil || isEmptyList(val) {
		return false
	}
	return !reflect.ValueOf(val).IsZero()
}

func isEmptyList(val any) bool {
	list, ok := val.([]any)
	return ok && len(list) == 0
}

func cloneValue(val any) any {
	switch val := val.(type) {
	case symbol:
		if val == "nil" {
			return nil
		}
		return val
	case []any:
		cloned := make([]any, len(val))
		for i, item := range val {
			cloned[i] = cloneValue(item)
		}
		return cloned
	default:
		return val
	}
}
