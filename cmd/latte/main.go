package main

import (
	"fmt"
	"log"
	"os"

	"github.com/igorcafe/latte"
	"github.com/lmorg/readline/v4"
)

func main() {
	env := latte.NewEnv()

	if len(os.Args) > 1 {
		for _, path := range os.Args[1:] {
			b, err := os.ReadFile(path)
			if err != nil {
				log.Printf("error reading file '%s': %v", path, err)
				continue
			}
			if _, err := env.Eval(string(b)); err != nil {
				log.Printf("error evaluating file '%s': %v", path, err)
			}
		}
		return
	}

	rl := readline.NewInstance()
	rl.SetPrompt("latte 🐶> ")

	for {
		func() {
			line, err := rl.Readline()
			if err != nil {
				os.Exit(1)
			}

			val, err := env.Eval(line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				return
			}
			fmt.Println("$", val)
		}()
	}
}
