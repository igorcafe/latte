package main

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"
)

func TestLatte(t *testing.T) {
	functions := maps.Clone(stdlibFunctions)
	functions["hey"] = func(args []any) any {
		if len(args) >= 1 {
			return args[0]
		}
		return "ho!"
	}
	functions["message"] = func(args []any) any {
		if len(args) == 0 {
			return nil
		}
		if len(args) == 1 {
			return args[0]
		}
		return fmt.Sprintf(args[0].(string), args[1:]...)
	}

	variables := maps.Clone(stdlibVariables)
	variables["ho"] = "I'm ho!"

	tests := []struct {
		source     string
		wantTokens []string
		wantAST    any
		wantEval   any
	}{
		{
			source:     "true",
			wantTokens: []string{"true"},
			wantAST:    Symbol("true"),
			wantEval:   true,
		},
		{
			source:     "\n\n\t  true \n\n\t",
			wantTokens: []string{"true"},
			wantAST:    Symbol("true"),
			wantEval:   true,
		},
		{
			source:     "nil",
			wantTokens: []string{"nil"},
			wantAST:    Symbol("nil"),
			wantEval:   nil,
		},
		{
			source:     `"text"`,
			wantTokens: []string{`"text"`},
			wantAST:    "text",
			wantEval:   "text",
		},
		{
			source:     `"text with spaces"`,
			wantTokens: []string{`"text with spaces"`},
			wantAST:    "text with spaces",
			wantEval:   "text with spaces",
		},
		{
			source:     "123",
			wantTokens: []string{"123"},
			wantAST:    123.0,
			wantEval:   123.0,
		},
		{
			source:     `()`,
			wantTokens: []string{"(", ")"},
			wantAST:    []any{},
		},
		{
			source:     `(hey)`,
			wantTokens: []string{"(", "hey", ")"},
			wantAST:    []any{Symbol("hey")},
			wantEval:   "ho!",
		},
		{
			source:     `(hey ho)`,
			wantTokens: []string{"(", "hey", "ho", ")"},
			wantAST:    []any{Symbol("hey"), Symbol("ho")},
			wantEval:   "I'm ho!",
		},
		{
			source:     `(message "hello!")`,
			wantTokens: []string{"(", "message", `"hello!"`, ")"},
			wantAST:    []any{Symbol("message"), "hello!"},
			wantEval:   "hello!",
		},
		{
			source:     "(+ 1 (+ 0 1))",
			wantTokens: []string{"(", "+", "1", "(", "+", "0", "1", ")", ")"},
			wantAST:    []any{Symbol("+"), 1.0, []any{Symbol("+"), 0.0, 1.0}},
			wantEval:   2.0,
		},
		{
			source:     "(* 5.5 3)",
			wantTokens: []string{"(", "*", "5.5", "3", ")"},
			wantAST:    []any{Symbol("*"), 5.5, 3.0},
			wantEval:   16.5,
		},
		{
			source:     "(/ 5 2)",
			wantTokens: []string{"(", "/", "5", "2", ")"},
			wantAST:    []any{Symbol("/"), 5.0, 2.0},
			wantEval:   2.5,
		},
		{
			source:     "(> 3 2 1)",
			wantTokens: []string{"(", ">", "3", "2", "1", ")"},
			wantAST:    []any{Symbol(">"), 3.0, 2.0, 1.0},
			wantEval:   true,
		},
		{
			source:     "(> 3 2 5)",
			wantTokens: []string{"(", ">", "3", "2", "5", ")"},
			wantAST:    []any{Symbol(">"), 3.0, 2.0, 5.0},
			wantEval:   false,
		},
		{
			source:     "(< 1 2 3)",
			wantTokens: []string{"(", "<", "1", "2", "3", ")"},
			wantAST:    []any{Symbol("<"), 1.0, 2.0, 3.0},
			wantEval:   true,
		},
		{
			source:     "(>= 3 3 2)",
			wantTokens: []string{"(", ">=", "3", "3", "2", ")"},
			wantAST:    []any{Symbol(">="), 3.0, 3.0, 2.0},
			wantEval:   true,
		},
		{
			source:     "(<= 1 1 2)",
			wantTokens: []string{"(", "<=", "1", "1", "2", ")"},
			wantAST:    []any{Symbol("<="), 1.0, 1.0, 2.0},
			wantEval:   true,
		},
	}

	for _, test := range tests {
		t.Run("expr: "+test.source, func(t *testing.T) {
			gotTokens := tokenize(test.source)
			if !slices.Equal(test.wantTokens, gotTokens) {
				t.Fatalf("\nTokens\n\nwant:\n%+#v\n\n\ngot:\n%+#v\n\n", test.wantTokens, gotTokens)
			}

			gotAST := parse(gotTokens)
			if !reflect.DeepEqual(gotAST, test.wantAST) {
				t.Fatalf("\nAST\n\nwant:\n%+#v (%T)\n\n\ngot:\n%+#v (%T)\n\n", test.wantAST, test.wantAST, gotAST, gotAST)
			}

			gotEval := eval(gotAST, functions, variables)
			if !reflect.DeepEqual(gotEval, test.wantEval) {
				t.Fatalf("\nEval\n\nwant:\n%+#v (%T)\n\n\ngot:\n%+#v (%T)\n\n", test.wantEval, test.wantEval, gotEval, gotEval)
			}
		})
	}
}
