package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/igxactly/yabtago/lib"
)

//#################
//# main program flow starts here
func main() {
	// runtime.GOMAXPROCS(2)

	// program arguments declaration
	var args struct {
		Action    string `arg:"positional,required,help:parse or report"`
		Input     string `arg:"positional,required,help:specify the input blktrace file"`
		Output    string `arg:"-o"`
		OutFormat string `arg:"--oformat,-f"`
		Config    string `arg:"-c,help:parse/report configuration"`
	}

	// default argument values
	args.OutFormat = "default"

	// parse arguments
	arg.MustParse(&args)
	// fmt.Printf("%+v\n", args)

	// try to open in/output files
	fileInput, err := os.Open(args.Input)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer fileInput.Close()
	reader := bufio.NewReader(fileInput)

	var config *lib.Config
	if args.Config != "" {
		configFile, err := os.Open(args.Config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer fileInput.Close()

		config = lib.NewConfig(configFile)
	} else {
		config = lib.DefaultConfig()
	}

	var writer *bufio.Writer
	if args.Output != "" {
		fileOutput, err := os.OpenFile(args.Output, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer fileOutput.Close()
		writer = bufio.NewWriter(fileOutput)
	}

	switch args.Action {
	case "parse":
		lib.Parse(reader, writer, config, args.OutFormat)
		break
	case "report":
		lib.Report(reader, writer, config, args.OutFormat)
		break
	default:
		fmt.Println("undefined action '" + args.Action + "'")
	}
}
