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
			env.Eval(string(b))
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

			val := env.Eval(line)
			fmt.Println("$", val)
		}()
	}
}
