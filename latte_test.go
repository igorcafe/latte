package latte

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func TestLatte(t *testing.T) {
	tests := []struct {
		source     string
		wantTokens []string
		wantAST    any
		wantEval   any
	}{
		{
			source:     "true",
			wantTokens: []string{"true"},
			wantAST:    symbol("true"),
			wantEval:   true,
		},
		{
			source:     "\n\n\t  true \n\n\t",
			wantTokens: []string{"true"},
			wantAST:    symbol("true"),
			wantEval:   true,
		},
		{
			source:     "nil",
			wantTokens: []string{"nil"},
			wantAST:    symbol("nil"),
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
			source:     `"text; with semicolon"`,
			wantTokens: []string{`"text; with semicolon"`},
			wantAST:    "text; with semicolon",
			wantEval:   "text; with semicolon",
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
			wantAST:    []any{symbol("hey")},
			wantEval:   "ho!",
		},
		{
			source:     `(hey ho)`,
			wantTokens: []string{"(", "hey", "ho", ")"},
			wantAST:    []any{symbol("hey"), symbol("ho")},
			wantEval:   "I'm ho!",
		},
		{
			source:     `(message "hello!")`,
			wantTokens: []string{"(", "message", `"hello!"`, ")"},
			wantAST:    []any{symbol("message"), "hello!"},
			wantEval:   "hello!",
		},
		{
			source:     "(+ 1 (+ 0 1))",
			wantTokens: []string{"(", "+", "1", "(", "+", "0", "1", ")", ")"},
			wantAST:    []any{symbol("+"), 1.0, []any{symbol("+"), 0.0, 1.0}},
			wantEval:   2.0,
		},
		{
			source:     "; ignore me\n(+ 1 2) ; ignore me too",
			wantTokens: []string{"(", "+", "1", "2", ")"},
			wantAST:    []any{symbol("+"), 1.0, 2.0},
			wantEval:   3.0,
		},
		{
			source:     "(* 5.5 3)",
			wantTokens: []string{"(", "*", "5.5", "3", ")"},
			wantAST:    []any{symbol("*"), 5.5, 3.0},
			wantEval:   16.5,
		},
		{
			source:     "(/ 5 2)",
			wantTokens: []string{"(", "/", "5", "2", ")"},
			wantAST:    []any{symbol("/"), 5.0, 2.0},
			wantEval:   2.5,
		},
		{
			source:     "(> 3 2 1)",
			wantTokens: []string{"(", ">", "3", "2", "1", ")"},
			wantAST:    []any{symbol(">"), 3.0, 2.0, 1.0},
			wantEval:   true,
		},
		{
			source:     "(> 3 2 5)",
			wantTokens: []string{"(", ">", "3", "2", "5", ")"},
			wantAST:    []any{symbol(">"), 3.0, 2.0, 5.0},
			wantEval:   false,
		},
		{
			source:     "(< 1 2 3)",
			wantTokens: []string{"(", "<", "1", "2", "3", ")"},
			wantAST:    []any{symbol("<"), 1.0, 2.0, 3.0},
			wantEval:   true,
		},
		{
			source:     "(>= 3 3 2)",
			wantTokens: []string{"(", ">=", "3", "3", "2", ")"},
			wantAST:    []any{symbol(">="), 3.0, 3.0, 2.0},
			wantEval:   true,
		},
		{
			source:     "(<= 1 1 2)",
			wantTokens: []string{"(", "<=", "1", "1", "2", ")"},
			wantAST:    []any{symbol("<="), 1.0, 1.0, 2.0},
			wantEval:   true,
		},
	}

	for _, test := range tests {
		t.Run("expr: "+test.source, func(t *testing.T) {
			env := testEnvWithFixtures()

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
		source  string
		want    any
		wantErr bool
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
		{source: "(quote hello)", want: symbol("hello")},
		{source: "(quote nil)", want: nil},
		{source: "(quote (1 2 3))", want: []any{1.0, 2.0, 3.0}},
		{source: "(quote (+ 1 unknown-symbol))", want: []any{symbol("+"), 1.0, symbol("unknown-symbol")}},
		{source: "(quote (1 (2 3)))", want: []any{1.0, []any{2.0, 3.0}}},
		{source: "'hello", want: symbol("hello")},
		{source: "'nil", want: nil},
		{source: "'(1 2 3)", want: []any{1.0, 2.0, 3.0}},
		{source: "'(+ 1 unknown-symbol)", want: []any{symbol("+"), 1.0, symbol("unknown-symbol")}},
		{source: "(list 'hello '(1 2))", want: []any{symbol("hello"), []any{1.0, 2.0}}},
		{source: "(if)", wantErr: true},
		{source: "(if unknown-symbol)", wantErr: true},
		{source: "(when)", wantErr: true},
		{source: "(unless)", wantErr: true},
		{source: "(quote)", wantErr: true},
		{source: "(quote 1 2)", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			got, err := testEnv().Eval(test.source)
			gotErr := err != nil
			if gotErr != test.wantErr {
				t.Fatalf("error: want %v, got %v (%v)", test.wantErr, gotErr, err)
			}
			if test.wantErr {
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("want %#v (%T), got %#v (%T)", test.want, test.want, got, got)
			}
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
		source  string
		want    any
		wantErr bool
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
		{source: "(nth 0 '(a b c))", want: symbol("a")},
		{source: "(nth 1 '(a b c))", want: symbol("b")},
		{source: "(nth 3 '(a b c))", want: nil},
		{source: "(nth 0 '())", want: nil},
		{source: "(nth 0 nil)", want: nil},
		{source: "(null nil)", want: true},
		{source: "(null '())", want: true},
		{source: "(null '(1))", want: false},
		{source: "(length '(1 2 3))", want: 3.0},
		{source: "(length '())", want: 0.0},
		{source: "(length nil)", want: 0.0},
		{source: "(car)", wantErr: true},
		{source: "(car 1)", wantErr: true},
		{source: "(car (quote (1 2)) 3)", wantErr: true},
		{source: "(cdr)", wantErr: true},
		{source: "(cdr 1)", wantErr: true},
		{source: "(cdr (quote (1 2)) 3)", wantErr: true},
		{source: "(nth)", wantErr: true},
		{source: "(nth 0)", wantErr: true},
		{source: "(nth 0 1)", wantErr: true},
		{source: "(nth \"0\" '(a))", wantErr: true},
		{source: "(nth -1 '(a))", wantErr: true},
		{source: "(nth 1.5 '(a))", wantErr: true},
		{source: "(null)", wantErr: true},
		{source: "(null nil nil)", wantErr: true},
		{source: "(length)", wantErr: true},
		{source: "(length 1)", wantErr: true},
		{source: "(length '(1) '(2))", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			got, err := testEnv().Eval(test.source)
			gotErr := err != nil
			if gotErr != test.wantErr {
				t.Fatalf("error: want %v, got %v (%v)", test.wantErr, gotErr, err)
			}
			if test.wantErr {
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("want %#v (%T), got %#v (%T)", test.want, test.want, got, got)
			}
		})
	}
}

func TestEvalLet(t *testing.T) {
	tests := []struct {
		source  string
		want    any
		wantErr bool
	}{
		{source: "(let () 1 2)", want: 2.0},
		{source: "(let ((x 2)) x)", want: 2.0},
		{source: "(let ((x 2) (y 3)) (+ x y))", want: 5.0},
		{source: "(let ((xs '(1 2 3))) (car xs))", want: 1.0},
		{source: "(let ((x 1)) (let ((y x)) y))", want: 1.0},
		{source: "(let ((x 1)) (let ((x 2)) x))", want: 2.0},
		{source: "(progn (define x 10) (let ((x 2)) x) x)", want: 10.0},
		{source: "(let ((x 1)))", want: nil},
		{source: "(progn (define x 10) (let ((x 1) (y x)) y))", want: 10.0},
		{source: "(let* () 1 2)", want: 2.0},
		{source: "(let* ((x 2)) x)", want: 2.0},
		{source: "(let* ((x 2) (y (+ x 3))) y)", want: 5.0},
		{source: "(progn (define x 10) (let* ((x 1) (y x)) y))", want: 1.0},
		{source: "(progn (define x 10) (let* ((x 1) (x (+ x 1))) x))", want: 2.0},
		{source: "(let* ((x '(1 2 3)) (y (cdr x))) (car y))", want: 2.0},
		{source: "(let* ((x 1)))", want: nil},
		{source: "(let 1 2)", wantErr: true},
		{source: "(let (x 1) x)", wantErr: true},
		{source: "(let ((x)) x)", wantErr: true},
		{source: "(let ((x 1 2)) x)", wantErr: true},
		{source: "(let ((1 2)) 3)", wantErr: true},
		{source: "(let ((x unknown-symbol)) x)", wantErr: true},
		{source: "(let* 1 2)", wantErr: true},
		{source: "(let* (x 1) x)", wantErr: true},
		{source: "(let* ((x)) x)", wantErr: true},
		{source: "(let* ((x 1 2)) x)", wantErr: true},
		{source: "(let* ((1 2)) 3)", wantErr: true},
		{source: "(let* ((x unknown-symbol)) x)", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			got, err := testEnv().Eval(test.source)
			gotErr := err != nil
			if gotErr != test.wantErr {
				t.Fatalf("error: want %v, got %v (%v)", test.wantErr, gotErr, err)
			}
			if test.wantErr {
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("want %#v (%T), got %#v (%T)", test.want, test.want, got, got)
			}
		})
	}
}

func TestEvalDefineFunction(t *testing.T) {
	tests := []struct {
		source  string
		want    any
		wantErr bool
	}{
		{source: "(progn (define (add1 x) (+ x 1)) (add1 2))", want: 3.0},
		{source: "(progn (define (add x y) (+ x y)) (add 2 3))", want: 5.0},
		{source: "(progn (define (answer) 42) (answer))", want: 42.0},
		{source: "(progn (define (f x) (+ x 1) (* x 2)) (f 3))", want: 6.0},
		{source: "(progn (define x 10) (define (add-x y) (+ x y)) (add-x 2))", want: 12.0},
		{source: "(progn (define (first xs) (car xs)) (first '(1 2 3)))", want: 1.0},
		{source: "(progn (define (noop)) (noop))", want: nil},
		{source: "(define)", wantErr: true},
		{source: "(define x)", wantErr: true},
		{source: "(define x 1 2)", wantErr: true},
		{source: "(define ())", wantErr: true},
		{source: "(define (1 x) x)", wantErr: true},
		{source: "(define (f 1) 1)", wantErr: true},
		{source: "(progn (define (f x) x) (f))", wantErr: true},
		{source: "(progn (define (f x) x) (f 1 2))", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			got, err := testEnv().Eval(test.source)
			gotErr := err != nil
			if gotErr != test.wantErr {
				t.Fatalf("error: want %v, got %v (%v)", test.wantErr, gotErr, err)
			}
			if test.wantErr {
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("want %#v (%T), got %#v (%T)", test.want, test.want, got, got)
			}
		})
	}
}

func TestEvalPredicates(t *testing.T) {
	tests := []struct {
		source  string
		want    any
		wantErr bool
	}{
		{source: "(nil? nil)", want: true},
		{source: "(nil? '())", want: false},
		{source: "(nil? false)", want: false},
		{source: "(list? '())", want: true},
		{source: "(list? '(1))", want: true},
		{source: "(list? nil)", want: false},
		{source: "(empty? '())", want: true},
		{source: "(empty? nil)", want: false},
		{source: "(empty? '(1))", want: false},
		{source: "(number? 1)", want: true},
		{source: `(number? "1")`, want: false},
		{source: `(string? "x")`, want: true},
		{source: "(string? 'x)", want: false},
		{source: "(symbol? 'x)", want: true},
		{source: `(symbol? "x")`, want: false},
		{source: "(bool? true)", want: true},
		{source: "(bool? false)", want: true},
		{source: "(bool? nil)", want: false},
		{source: "(nil?)", wantErr: true},
		{source: "(list? '() '())", wantErr: true},
		{source: "(empty?)", wantErr: true},
		{source: "(number?)", wantErr: true},
		{source: "(string?)", wantErr: true},
		{source: "(symbol?)", wantErr: true},
		{source: "(bool?)", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			got, err := testEnv().Eval(test.source)
			gotErr := err != nil
			if gotErr != test.wantErr {
				t.Fatalf("error: want %v, got %v (%v)", test.wantErr, gotErr, err)
			}
			if test.wantErr {
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("want %#v (%T), got %#v (%T)", test.want, test.want, got, got)
			}
		})
	}
}

func TestParseProgram(t *testing.T) {
	tests := []struct {
		source string
		want   any
	}{
		{
			source: `
				(define x 1)
				(define y 2)
				(+ x y)
			`,
			want: 3.0,
		},
		{source: "(define x 1) (define y 2) (+ x y)", want: 3.0},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			got, err := testEnv().Eval(test.source)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("want %#v (%T), got %#v (%T)", test.want, test.want, got, got)
			}
		})
	}
}

func TestEnvPublicAPI(t *testing.T) {
	env := NewEnv()
	env.Define("x", 10.0)
	env.RegisterFunction("double", func(args []any) any {
		return args[0].(float64) * 2
	})

	got, err := env.Eval("(double x)")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, 20.0) {
		t.Fatalf("want %#v (%T), got %#v (%T)", 20.0, 20.0, got, got)
	}
}

func evalSource(t *testing.T, source string) any {
	t.Helper()

	got, err := testEnv().Eval(source)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func testEnv() *Env {
	return NewEnv()
}

func testEnvWithFixtures() *Env {
	env := testEnv()
	env.RegisterFunction("hey", func(args []any) any {
		if len(args) >= 1 {
			return args[0]
		}
		return "ho!"
	})
	env.RegisterFunction("message", func(args []any) any {
		if len(args) == 0 {
			return nil
		}
		if len(args) == 1 {
			return args[0]
		}
		return fmt.Sprintf(args[0].(string), args[1:]...)
	})
	env.Define("ho", "I'm ho!")
	return env
}
