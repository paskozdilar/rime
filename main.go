package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/paskozdilar/rime/src/rime"
)

func main() {
	var word string
	var syllables int
	var err error

	flag.Parse()
	if syllables, err = strconv.Atoi(flag.Arg(1)); flag.NArg() != 2 || len(flag.Arg(0)) == 0 || err != nil {
		fmt.Println("Usage:\n\trime WORD SYLLABLES")
		os.Exit(0)
	}
	word = flag.Arg(0)

	r := rime.NewRime(word, syllables)
	defer r.Close()
	for word := range r.Channel() {
		fmt.Println(word)
	}
}
