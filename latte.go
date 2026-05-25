package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/lmorg/readline/v4"
)

func main() {
	log.Default().SetOutput(io.Discard)

	functions := map[Symbol]func([]any) any{
		//
		"hey": func(args []any) any {
			if len(args) >= 1 {
				return args[0]
			}
			return "ho!"
		},
		"message": func(args []any) any {
			if len(args) == 0 {
				return nil
			}
			if len(args) == 1 {
				return args[0]
			}
			return fmt.Sprintf(args[0].(string), args[1:]...)
		},
		"+": func(args []any) any {
			total := args[0].(float64)
			for _, arg := range args[1:] {
				total -= arg.(float64)
			}
			return total
		},
		"-": func(args []any) any {
			total := args[0].(float64)
			for _, arg := range args[1:] {
				total -= arg.(float64)
			}
			return total
		},
		"*": func(args []any) any {
			total := args[0].(float64)
			for _, arg := range args[1:] {
				total *= arg.(float64)
			}
			return total
		},
		"/": func(args []any) any {
			total := args[0].(float64)
			for _, arg := range args[1:] {
				total /= arg.(float64)
			}
			return total
		},
	}

	variables := map[Symbol]any{
		"ho":  "I'm ho!",
		"t":   true,
		"nil": nil,
	}

	rl := readline.NewInstance()
	rl.SetPrompt("latte 🐶> ")

	for {
		line, err := rl.Readline()

		if err != nil {
			return
		}

		tokens := tokenize(line)
		ast := parse(tokens)
		val := eval(ast, functions, variables)
		fmt.Println("$", val)
	}
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
			// salvar localizacao da lista atual na stack?
			// adicionar uma lista na posicao atual e passar a trabalhar nela
			// "(", "hello", "(", "world", ")", ")"
			// []any{"hello", []any{"world"}}
			next := []any{}
			tree = append(tree, next)
			stack = append(stack, tree)
			tree = next

			// next := []any{}
			// curr = append(curr, next)
			// stack = append(stack, len(tree)-1)
			// curr = append(curr, next)
			// prev = curr
			// curr = next
			log.Print("parser: stack pushed")
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

			// next := []any{}
			// curr = append(curr, next)
			// prev = curr
			// curr = next
			log.Print("parser: stack poped")
			continue
		}
		log.Printf("parser: current token: %s", token)
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

		// reflectFn := reflect.ValueOf(fn)

		// if !reflectFn.IsValid() {
		// 	panic("TODO: symbol is not a valid function: " + string(symbol))
		// }

		// if reflectFn.Kind() != reflect.Func {
		// 	panic("TODO: symbol is not a function: " + string(symbol))
		// }

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
