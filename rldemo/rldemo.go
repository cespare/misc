package main

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/chzyer/readline"
)

func main() {
	l, err := readline.NewEx(&readline.Config{
		Prompt:      "> ",
		HistoryFile: "/tmp/rldemo.txt",
		//VimMode: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		line, err := l.Readline()
		switch err {
		case nil:
		case readline.ErrInterrupt:
			continue
		case io.EOF:
			fmt.Println("EOF")
			return
		default:
			log.Println("Readline error:", err)
			continue
		}
		time.Sleep(250 * time.Millisecond)
		if line == "" {
			continue
		}
		fmt.Printf("you typed %q\n", line)
	}
}
