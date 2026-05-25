package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/lmorg/readline/v4"
)

type Env struct {
	Functions map[Symbol]func([]any) any
	Variables map[Symbol]any
	Parent    *Env
}

func (env *Env) FindSymbol(symbol Symbol) (any, bool) {
	if variable, ok := env.Variables[symbol]; ok {
		return variable, ok
	}
	if env.Parent != nil {
		return env.Parent.FindSymbol(symbol)
	}
	return nil, false
}

func (env *Env) DefineVariable(name Symbol, value any) {
	env.Variables[name] = value
}

func (env *Env) ChildScope() *Env {
	return &Env{
		Functions: env.Functions,
		Variables: make(map[Symbol]any),
		Parent:    env,
	}
}

func main() {
	env := &Env{
		Functions: stdlibFunctions,
		Variables: stdlibVariables,
		Parent:    nil,
	}

	if len(os.Args) > 1 {
		for _, path := range os.Args[1:] {
			b, err := os.ReadFile(path)
			if err != nil {
				log.Printf("error reading file '%s': %v", path, err)
			}
			evalProgram(string(b), env)
		}
		return
	}

	rl := readline.NewInstance()
	rl.SetPrompt("latte 🐶> ")

	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "panic: %v", r)
				}
			}()

			line, err := rl.Readline()

			if err != nil {
				os.Exit(1)
			}

			val := evalProgram(line, env)
			fmt.Println("$", val)
		}()
	}
}

var stdlibVariables = map[Symbol]any{
	"true":  true,
	"false": false,
	"nil":   nil,
}

var stdlibFunctions = map[Symbol]func(args []any) any{
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

type Symbol string

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
		return []any{Symbol("quote"), parseExpr(tokens, pos)}
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
	return Symbol(token)
}

func eval(node any, env *Env) any {
	switch node := node.(type) {
	case Symbol:
		value, ok := env.FindSymbol(node)
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

		symbol := node[0].(Symbol)

		// special functions
		switch symbol {
		case "let", "let*":
			if len(node) == 1 {
				return nil
			}
			bindings, ok := node[1].([]any)
			if !ok {
				panic("let expression expected bindings at position 2")
			}

			scope := env.ChildScope()
			for _, binding := range bindings {
				binding, ok := binding.([]any)
				if !ok || len(binding) != 2 {
					panic("let expression invalid binding")
				}
				name, ok := binding[0].(Symbol)
				if !ok {
					panic("let binding is not a symbol: " + fmt.Sprint(binding[0]))
				}

				if string(symbol) == "let" {
					scope.DefineVariable(name, eval(binding[1], env))
				} else {
					scope.DefineVariable(name, eval(binding[1], scope))
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
			name, ok := node[1].(Symbol)
			if !ok {
				panic("define argument 1 must be a symbol")
			}
			value := eval(node[2], env)
			env.DefineVariable(name, value)
			return value

		case "when", "unless":
			if len(node) < 2 {
				panic(string(symbol) + " needs a condition")
			}

			cond := eval(node[1], env)
			if isTruthy(cond) == (string(symbol) == "unless") {
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

		function, ok := env.Functions[symbol]
		if !ok {
			panic("unknown function " + string(symbol))
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

func evalProgram(source string, env *Env) any {
	var result any
	for _, ast := range parseProgram(tokenize(source)) {
		result = eval(ast, env)
	}
	return result
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
	case Symbol:
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
