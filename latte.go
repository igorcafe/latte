package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/lmorg/readline/v4"
)

func main() {
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

			tokens := tokenize(line)
			ast := parse(tokens)
			val := eval(ast, stdlibFunctions, stdlibVariables)
			fmt.Println("$", val)
		}()
	}
}

var stdlibVariables = map[Symbol]any{
	"t":   true,
	"f":   false,
	"nil": nil,
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
		return reflect.ValueOf(args[0]).IsZero()
	},
	"=": lispEqual,
	"!=": func(args []any) any {
		return !lispEqual(args).(bool)
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

		if len(args) == 1 {
			fmt.Printf(args[0].(string))
			return nil
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
		if slices.Contains([]rune{'(', ')'}, r) {
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

	if len(tokens) == 1 {
		var value any
		if err := json.Unmarshal([]byte(tokens[0]), &value); err == nil {
			switch value.(type) {
			case string, int, float64:
				// avoid converting "true", "null" and things like that
				return value
			}
		}
		return Symbol(tokens[0])
	}

	if tokens[0] != "(" || tokens[len(tokens)-1] != ")" {
		panic("TODO: syntax error, unbalanced parenthesis")
	}

	tokens = tokens[1 : len(tokens)-1]

	tree := []any{}
	stack := [][]any{}

	for _, token := range tokens {
		if token == "(" {
			next := []any{}
			tree = append(tree, next)
			stack = append(stack, tree)
			tree = next

			continue
		}
		if token == ")" {
			if len(stack) == 0 {
				panic("TODO: error unbalanced parenthesis")
			}

			curr := tree
			tree = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			tree[len(tree)-1] = curr

			continue
		}
		tree = append(tree, parse([]string{token}))
	}

	return tree
}

func eval(node any, functions map[Symbol]func([]any) any, variables map[Symbol]any) any {
	switch node := node.(type) {
	case Symbol:
		value, ok := variables[node]
		if !ok {
			value, ok := functions[node]
			if !ok {
				panic("unknown symbol: " + string(node))
			}
			return value
		}
		return value
	case string, int, float64:
		return node
	case []any:
		if len(node) == 0 {
			return nil
		}

		symbol := node[0].(Symbol)
		if _, ok := functions[symbol]; !ok {
			panic("TODO: unknown function " + string(symbol))
		}

		fnVal := reflect.ValueOf(functions[symbol])

		args := []any{}
		if len(node) > 1 {
			for _, node := range node[1:] {
				arg := eval(node, functions, variables)
				args = append(args, arg)
			}
		}

		results := fnVal.Call([]reflect.Value{reflect.ValueOf(args)})
		if len(results) > 1 {
			panic("TODO: lisp exposed functions should return a single value")
		}
		if len(results) == 0 {
			return nil
		}
		return results[0].Interface()
	}

	return nil
}
