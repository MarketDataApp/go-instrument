package main

import (
	"errors"
	"flag"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"

	"github.com/MarketDataApp/go-instrument/instrument"
	"github.com/MarketDataApp/go-instrument/processor"
)

func main() {
	var (
		fileName      string
		overwrite     bool
		defaultSelect bool
		skipGenerated bool
	)
	flag.StringVar(&fileName, "filename", "", "go file to instrument")
	flag.BoolVar(&overwrite, "w", false, "overwrite original file")
	flag.BoolVar(&defaultSelect, "all", true, "instrument all by default")
	flag.BoolVar(&skipGenerated, "skip-generated", false, "skip generated files")
	flag.Parse()

	if err := process(fileName, overwrite, defaultSelect, skipGenerated); err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}

func process(fileName string, overwrite, defaultSelect, skipGenerated bool) error {
	if fileName == "" {
		return errors.New("missing arg: file name")
	}

	src, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	formattedSrc, err := format.Source(src)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, fileName, formattedSrc, parser.ParseComments)
	if err != nil {
		return err
	}
	if skipGenerated && ast.IsGenerated(file) {
		log.Printf("skipping generated file: %s\n", fileName)
		return nil
	}

	directives := processor.GoBuildDirectivesFromFile(*file)
	for _, q := range directives {
		if q.SkipFile() {
			return nil
		}
	}

	commands, err := processor.CommandsFromFile(*file)
	if err != nil {
		return err
	}
	functionSelector := processor.NewMapFunctionSelectorFromCommands(defaultSelect, commands)

	// Extract package name
	packageName := file.Name.Name

	var instrumenter processor.Instrumenter = &instrument.Sentry{
		TracerName:  packageName,
		ContextName: "ctx",
		ErrorName:   "err",
	}
	p := processor.Processor{
		Instrumenter:     instrumenter,
		FunctionSelector: functionSelector,
		SpanName:         processor.BasicSpanName,
		ContextName:      "ctx",
		ContextPackage:   "context",
		ContextType:      "Context",
		ErrorName:        "err",
		ErrorType:        `error`,
	}

	if err := p.Process(fset, file); err != nil {
		return err
	}

	var out io.Writer = os.Stdout
	if overwrite {
		outf, err := os.OpenFile(fileName, os.O_RDWR|os.O_TRUNC, 0)
		if err != nil {
			return err
		}
		defer outf.Close()
		out = outf
	}

	return format.Node(out, fset, file)
}