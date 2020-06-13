package main

import (
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"

	"github.com/roy2220/proxyz/cmd/proxyz/internal/methodset"
	"github.com/roy2220/proxyz/cmd/proxyz/internal/proxygen"
)

func main() {
	var args struct {
		InputPackagePath  string `arg:"positional" help:"input package path in Go" placeholder:"IPKG"`
		InputTypeName     string `arg:"positional" help:"input type name in Go" placeholder:"ITYPE"`
		OutputPackagePath string `arg:"positional" help:"output package path in Go" placeholder:"OPKG"`
		OutputTypeName    string `arg:"positional" help:"output type name in Go" placeholder:"OTYPE"`
		FormatOutput      bool   `arg:"-f,--format" default:"true" help:"format output"`
		OutputFileName    string `arg:"-w,--write" help:"write output to file inside output package directory" placeholder:"FILE"`
	}

	arg.MustParse(&args)

	if args.OutputFileName != "" {
		if filepath.Ext(args.OutputFileName) != ".go" {
			fatalf("%q is not go file name", args.OutputFileName)
		}

		if filepath.Base(args.OutputFileName) != args.OutputFileName {
			fatalf("%q should not contain directory path", args.OutputFileName)
		}
	}

	parseContext := new(methodset.ParseContext).Init()
	var methodSet methodset.MethodSet

	if err := methodSet.ParseType(parseContext, args.InputPackagePath, args.InputTypeName); err != nil {
		fatal(err)
	}

	proxyGen := proxygen.ProxyGen{
		MethodSet:         &methodSet,
		OutputPackagePath: args.OutputPackagePath,
		OutputTypeName:    args.OutputTypeName,
	}

	output, err := proxyGen.EmitProgram()

	if err != nil {
		fatal(err)
	}

	if args.FormatOutput {
		output, err = format.Source(output)

		if err != nil {
			fatalf("failed to format output: %v", err)
		}
	}

	if args.OutputFileName != "" {
		outputFilePath := filepath.Join(proxyGen.OutputPackageDirPath(), args.OutputFileName)

		if err := ioutil.WriteFile(outputFilePath, output, 0644); err != nil {
			fatalf("failed to write output to file %q: %v", outputFilePath, err)
		}

		infof("output written to file %q", outputFilePath)
	} else {
		if _, err := os.Stdout.Write(output); err != nil {
			fatalf("failed to write output: %v", err)
		}
	}
}

func info(arg interface{}) {
	fmt.Fprintf(os.Stderr, "%v\n", arg)
}

func infof(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func fatal(arg interface{}) {
	info(arg)
	os.Exit(1)
}

func fatalf(format string, args ...interface{}) {
	infof(format, args)
	os.Exit(1)
}
