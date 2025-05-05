package main

import (
	conf "blanktiger/hm/configuration"
	"blanktiger/hm/instructions"
	"blanktiger/hm/lib"
	"os"
)

func main() {
	c := conf.Parse()
	c.Display()
	c.AssertCorrectness()

	lib.Logger = c.Logger
	instructions.Init(c.Logger)
	err := _main(&c)
	if err != nil {
		c.Logger.Error("program exited with an error", "error", err)
		os.Exit(1)
	}
}

func _main(c *conf.Configuration) error {
	if c.Tui {
		return tuiMain(c)
	} else {
		return cliMain(c)
	}
}
