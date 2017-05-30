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
	// program arguments declaration
	var args struct {
		Action       string `arg:"positional,required,help:parse or report"`
		Input        string `arg:"positional,required,help:specify the input blktrace file"`
		Output       string `arg:"-o"`
		OutFormat    string `arg:"--oformat,-f"`
		ReportConfig string `arg:"-r"`
	}

	// default argument values
	args.OutFormat = "default"

	// parse arguments
	arg.MustParse(&args)
	fmt.Printf("%+v\n", args)

	// try to open in/output files
	fileInput, err := os.Open(args.Input)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer fileInput.Close()
	reader := bufio.NewReader(fileInput)

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
		lib.Parse(reader, writer)
		break
	case "report":
		// TODO: Check report config and than go on
		lib.Report(reader, writer)
		break
	default:
		fmt.Println("undefined action '" + args.Action + "'")
	}
}
