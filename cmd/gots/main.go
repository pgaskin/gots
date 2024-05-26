// Command gots passes a font through OTS.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/pgaskin/gots"
)

var (
	Quiet   = flag.Bool("quiet", false, "do not output errors and warnings")
	Verbose = flag.Bool("verbose", false, "output information about processed files")
	Output  = flag.String("output", "", "set the output basename (instead of stdout)")
)

func main() {
	var err error

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [options] [-|input_file]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}

	var file string
	if file = flag.Arg(0); file == "" || file == "-" {
		file = "stdin"
	}

	var input []byte
	switch fn := flag.Arg(0); fn {
	case "", "-":
		input, err = io.ReadAll(os.Stdin)
	default:
		input, err = os.ReadFile(fn)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: fatal: %v\n", file, err)
		os.Exit(1)
	}

	var opt []gots.Option
	if !*Quiet {
		opt = append(opt, gots.WithMessages(func(level gots.MessageLevel, msg string) {
			switch level {
			case gots.MessageLevelError:
				fmt.Fprintf(os.Stderr, "%s: error: %s\n", file, msg)
			case gots.MessageLevelWarning:
				fmt.Fprintf(os.Stderr, "%s: warning: %s\n", file, msg)
			}
		}))
	}

	output, err := gots.Process(input, opt...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: fatal: %v\n", file, err)
		os.Exit(1)
	}
	ext := gots.Extension(output)

	if *Output == "-" {
		*Output = ""
	}
	if *Verbose {
		fmt.Fprintf(os.Stderr, "%s: writing %s (length=%d, input_length=%d, input_type=%s)\n", file, *Output+ext, len(output), len(input), gots.Extension(input))
	}
	if *Output == "" {
		_, err = os.Stdout.Write(output)
	} else {
		err = os.WriteFile(*Output+ext, output, 0666)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: fatal: %v\n", file, err)
		os.Exit(1)
	}
}
