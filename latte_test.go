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
			env := Env{
				Functions: functions,
				Variables: variables,
			}

			gotTokens := tokenize(test.source)
			if !slices.Equal(test.wantTokens, gotTokens) {
				t.Fatalf("\nTokens\n\nwant:\n%+#v\n\n\ngot:\n%+#v\n\n", test.wantTokens, gotTokens)
			}

			gotAST := parse(gotTokens)
			if !reflect.DeepEqual(gotAST, test.wantAST) {
				t.Fatalf("\nAST\n\nwant:\n%+#v (%T)\n\n\ngot:\n%+#v (%T)\n\n", test.wantAST, test.wantAST, gotAST, gotAST)
			}

			gotEval := eval(gotAST, env)
			if !reflect.DeepEqual(gotEval, test.wantEval) {
				t.Fatalf("\nEval\n\nwant:\n%+#v (%T)\n\n\ngot:\n%+#v (%T)\n\n", test.wantEval, test.wantEval, gotEval, gotEval)
			}
		})
	}
}

func TestEvalSpecialForms(t *testing.T) {
	tests := []struct {
		source string
		want   any
	}{
		{source: "(if true 1 2)", want: 1.0},
		{source: "(if false 1 2)", want: 2.0},
		{source: "(if nil 1 2)", want: 2.0},
		{source: "(if 0 1 2)", want: 2.0},
		{source: `(if "" 1 2)`, want: 2.0},
		{source: "(if true 1 unknown-symbol)", want: 1.0},
		{source: "(if false unknown-symbol 2)", want: 2.0},
		{source: "(if true)", want: nil},
		{source: "(when true 1 2)", want: 2.0},
		{source: "(when false unknown-symbol)", want: nil},
		{source: "(when nil unknown-symbol)", want: nil},
		{source: "(unless false 1 2)", want: 2.0},
		{source: "(unless nil 1 2)", want: 2.0},
		{source: "(unless true unknown-symbol)", want: nil},
		{source: "(progn)", want: nil},
		{source: "(progn 1 2 3)", want: 3.0},
		{source: "(quote hello)", want: Symbol("hello")},
		{source: "(quote nil)", want: nil},
		{source: "(quote (1 2 3))", want: []any{1.0, 2.0, 3.0}},
		{source: "(quote (+ 1 unknown-symbol))", want: []any{Symbol("+"), 1.0, Symbol("unknown-symbol")}},
		{source: "(quote (1 (2 3)))", want: []any{1.0, []any{2.0, 3.0}}},
		{source: "'hello", want: Symbol("hello")},
		{source: "'nil", want: nil},
		{source: "'(1 2 3)", want: []any{1.0, 2.0, 3.0}},
		{source: "'(+ 1 unknown-symbol)", want: []any{Symbol("+"), 1.0, Symbol("unknown-symbol")}},
		{source: "(list 'hello '(1 2))", want: []any{Symbol("hello"), []any{1.0, 2.0}}},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			got := evalSource(t, test.source)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("want %#v (%T), got %#v (%T)", test.want, test.want, got, got)
			}
		})
	}
}

func TestEvalSpecialFormsPanics(t *testing.T) {
	tests := []string{
		"(if)",
		"(if unknown-symbol)",
		"(when)",
		"(unless)",
		"(quote)",
		"(quote 1 2)",
	}

	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()

			evalSource(t, source)
		})
	}
}

func TestNilAndEmptyListSemantics(t *testing.T) {
	tests := []struct {
		source string
		want   any
	}{
		{source: "nil", want: nil},
		{source: "(quote nil)", want: nil},
		{source: "(quote ())", want: []any{}},
		{source: "(list)", want: []any{}},
		{source: "(= nil (quote ()))", want: false},
		{source: "(= nil (list))", want: false},
		{source: "(= (quote ()) (list))", want: true},
		{source: "(not nil)", want: true},
		{source: "(not (quote ()))", want: true},
		{source: "(if nil 1 2)", want: 2.0},
		{source: "(if (quote ()) 1 2)", want: 2.0},
		{source: "(car (quote nil))", want: nil},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			got := evalSource(t, test.source)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("want %#v (%T), got %#v (%T)", test.want, test.want, got, got)
			}
		})
	}
}

func TestEvalListFunctions(t *testing.T) {
	tests := []struct {
		source string
		want   any
	}{
		{source: "(list 1 2 3)", want: []any{1.0, 2.0, 3.0}},
		{source: "(list)", want: []any{}},
		{source: "(car (quote (1 2 3)))", want: 1.0},
		{source: "(car (list 1 2 3))", want: 1.0},
		{source: "(car nil)", want: nil},
		{source: "(car (quote nil))", want: nil},
		{source: "(car (quote ()))", want: nil},
		{source: "(cdr (quote (1 2 3)))", want: []any{2.0, 3.0}},
		{source: "(cdr (list 1 2 3))", want: []any{2.0, 3.0}},
		{source: "(cdr (quote (1)))", want: nil},
		{source: "(cdr nil)", want: nil},
		{source: "(cdr (quote nil))", want: nil},
		{source: "(cdr (quote ()))", want: nil},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			got := evalSource(t, test.source)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("want %#v (%T), got %#v (%T)", test.want, test.want, got, got)
			}
		})
	}
}

func TestEvalListFunctionsPanics(t *testing.T) {
	tests := []string{
		"(car)",
		"(car 1)",
		"(car (quote (1 2)) 3)",
		"(cdr)",
		"(cdr 1)",
		"(cdr (quote (1 2)) 3)",
	}

	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()

			evalSource(t, source)
		})
	}
}

func evalSource(t *testing.T, source string) any {
	t.Helper()

	functions := maps.Clone(stdlibFunctions)
	variables := maps.Clone(stdlibVariables)
	env := Env{
		Functions: functions,
		Variables: variables,
	}

	return eval(parse(tokenize(source)), env)
}
