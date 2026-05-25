# Latte

A small Lisp written in Go, with an embeddable API and a CLI with a REPL.

## Lisp

```lisp
(define tax-rate 0.1)

(define (line-total price quantity)
  (* price quantity))

(define (checkout subtotal discount)
  (let ((tax (* subtotal tax-rate)))
    (- (+ subtotal tax) discount)))

(println "total:" (checkout (line-total 25 3) 10))
```

## Go

```go
package main

import (
	"fmt"
	"log"

	"github.com/igorcafe/latte"
)

func main() {
	env := latte.NewEnv()

	env.Define("name", "Latte")
	env.RegisterFunction("shout", func(args []any) any {
		return fmt.Sprintf("%s!", args[0])
	})

	value, err := env.Eval(`(shout name)`)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(value)
}
```

## CLI

Install:

```sh
go install github.com/igorcafe/latte/cmd/latte@latest
```

Use as a REPL:

```sh
latte
```

Run files:

```sh
latte example.lisp
```
