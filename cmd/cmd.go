package cmd

import (
	"fmt"
	"os"
	"sort"
)

type Command struct {
	Use   string
	Short string
	Long  string

	run func(args []string) error

	subcommands map[string]*Command
}

func (c *Command) AddCommand(cmds ...*Command) {
	if c.subcommands == nil {
		c.subcommands = make(map[string]*Command)
	}
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		c.subcommands[cmd.Use] = cmd
	}
}

func (c *Command) Execute() error {
	return c.execute(os.Args[1:])
}

func (c *Command) execute(args []string) error {
	if len(args) == 0 {
		if c.run != nil {
			return c.run(args)
		}
		return c.usage()
	}

	switch args[0] {
	case "-h", "--help", "help":
		return c.usage()
	}

	if sub, ok := c.subcommands[args[0]]; ok {
		return sub.execute(args[1:])
	}

	if c.run != nil {
		return c.run(args)
	}

	return fmt.Errorf("unknown command %q", args[0])
}

func (c *Command) usage() error {
	message := c.Long
	if message == "" {
		message = c.Short
	}
	if message == "" {
		message = c.Use
	}
	fmt.Fprintln(os.Stdout, message)
	if len(c.subcommands) > 0 {
		fmt.Fprintln(os.Stdout, "\nAvailable commands:")
		names := make([]string, 0, len(c.subcommands))
		for name := range c.subcommands {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			cmd := c.subcommands[name]
			fmt.Fprintf(os.Stdout, "  %s\t%s\n", cmd.Use, cmd.Short)
		}
	}
	return nil
}

var RootCmd = &Command{
	Use:   "kg-service",
	Short: "KG Service",
	Long:  "KG Service orchestrating multi-tenant knowledge graph workflows",
}

func Execute() error {
	return RootCmd.Execute()
}
