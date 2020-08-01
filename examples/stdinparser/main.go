// example_1 parses StdIn as JSON input to tokens and dumps those to StdOut

package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gibsn/gojsonlex"
)

func main() {
	l, err := gojsonlex.NewJSONLexer(os.Stdin)
	if err != nil {
		log.Fatalf("fatal: could not create JSONLexer: %v", err)
	}

	for {
		l, err := l.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("fatal: could not parse input: %v", err)
		}

		fmt.Println(l)
	}
}
