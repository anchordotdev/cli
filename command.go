package cli

import (
	"context"
	"flag"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/joeshaw/envdecode"
	"github.com/mcuadros/go-defaults"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Command struct {
	TUI

	Name        string
	Use         string
	Short, Long string

	Group string

	SubCommands []*Command
}

func (c *Command) Execute(ctx context.Context, cfg *Config) error {
	defaults.SetDefaults(cfg)

	// enable ANSI processing for Windows, see: https://github.com/muesli/termenv#platform-support
	restoreConsole, err := termenv.EnableVirtualTerminalProcessing(termenv.DefaultOutput())
	if err != nil {
		panic(err)
	}
	defer restoreConsole()

	cmd := c.cobraCommand(ctx, reflect.ValueOf(cfg))

	if err := envdecode.Decode(cfg); err != nil && err != envdecode.ErrNoTargetFieldsAreSet {
		return err
	}

	return cmd.ExecuteContext(ctx)
}

func (c *Command) cobraCommand(ctx context.Context, cfgv reflect.Value) *cobra.Command {
	cmd := &cobra.Command{
		Use:     c.Use,
		Short:   c.Short,
		Long:    c.Long,
		GroupID: c.Group,
	}

	if c.Run != nil {
		cmd.RunE = func(_ *cobra.Command, args []string) error {
			// TODO: positional args

			return c.Run(ctx, termenv.DefaultOutput().TTY())
		}
	}

	cmdVals := make(map[string]reflect.Value)
	c.cobraBuild(cmd, cfgv, cmdVals)

	for _, subc := range c.SubCommands {
		val, ok := cmdVals[subc.Name]
		if !ok {
			panic("impossible")
		}

		cmd.AddCommand(subc.cobraCommand(ctx, val))
	}
	return cmd
}

func (c *Command) cobraBuild(cmd *cobra.Command, cfgv reflect.Value, cmdVals map[string]reflect.Value) {
	flags := cmd.Flags()
	if c.TUI.Run == nil {
		flags = cmd.PersistentFlags()
	}

	cfgi := reflect.Indirect(cfgv)
	cfgt := cfgi.Type()

	for i := 0; i < cfgt.NumField(); i++ {
		field := cfgt.Field(i)

		var usage string
		if desc, ok := field.Tag.Lookup("desc"); ok {
			usage = desc
		}

		if flag, ok := field.Tag.Lookup("flag"); ok {
			parts := strings.Split(flag, ",")
			name := parts[0]

			var shorthand string
			if len(parts) > 1 {
				shorthand = parts[1]
			}

			pval := reflect.Indirect(cfgi.Field(i))
			bindFlag(flags, pval.Addr().Interface(), name, shorthand, usage)
		}

		if group, ok := field.Tag.Lookup("group"); ok {
			parts := strings.Split(group, ",")
			id := parts[0]

			var title string
			if len(parts) > 1 {
				title = parts[1]
			}

			cmd.AddGroup(&cobra.Group{ID: id, Title: title})

			c.cobraBuild(cmd, cfgi.Field(i), cmdVals)
		} else if cmdName, ok := field.Tag.Lookup("cmd"); ok {
			cmdVals[cmdName] = cfgi.Field(i)
		} else if field.Type.Kind() == reflect.Struct {
			c.cobraBuild(cmd, cfgi.Field(i), cmdVals)
		}
	}
}

// github.com/AdamSLevy/flagbind
func bindFlag(fs *pflag.FlagSet, p interface{}, name, shortName, usage string) {

	var f *pflag.Flag
	switch p := p.(type) {
	case flag.Value:
		// Check if p also implements pflag.Value...
		pp, ok := p.(pflag.Value)
		if !ok {
			// If not, use the pflagValue shim...
			panic("TODO")
		}
		f = fs.VarPF(pp, name, shortName, usage)
	case *net.IP:
		val := *p
		fs.IPVarP(p, name, shortName, val, usage)
	case *[]net.IP:
		val := *p
		fs.IPSliceVarP(p, name, shortName, val, usage)
	case *bool:
		val := *p
		fs.BoolVarP(p, name, shortName, val, usage)
	case *[]bool:
		val := *p
		fs.BoolSliceVarP(p, name, shortName, val, usage)
	case *time.Duration:
		val := *p
		fs.DurationVarP(p, name, shortName, val, usage)
	case *[]time.Duration:
		val := *p
		fs.DurationSliceVarP(p, name, shortName, val, usage)
	case *int:
		val := *p
		fs.IntVarP(p, name, shortName, val, usage)
	case *[]int:
		val := *p
		fs.IntSliceVarP(p, name, shortName, val, usage)
	case *uint:
		val := *p
		fs.UintVarP(p, name, shortName, val, usage)
	case *[]uint:
		val := *p
		fs.UintSliceVarP(p, name, shortName, val, usage)
	case *int64:
		val := *p
		fs.Int64VarP(p, name, shortName, val, usage)
	case *[]int64:
		val := *p
		fs.Int64SliceVarP(p, name, shortName, val, usage)
	case *uint64:
		val := *p
		fs.Uint64VarP(p, name, shortName, val, usage)
	case *float32:
		val := *p
		fs.Float32VarP(p, name, shortName, val, usage)
	case *[]float32:
		val := *p
		fs.Float32SliceVarP(p, name, shortName, val, usage)
	case *float64:
		val := *p
		fs.Float64VarP(p, name, shortName, val, usage)
	case *[]float64:
		val := *p
		fs.Float64SliceVarP(p, name, shortName, val, usage)
	case *string:
		val := *p
		fs.StringVarP(p, name, shortName, val, usage)
	case *[]string:
		val := *p
		fs.StringSliceVarP(p, name, shortName, val, usage)
	default:
		panic("TODO")
	}

	if f == nil {
		f = fs.Lookup(name)
	}

	return
}
