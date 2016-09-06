package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/isaacev/Plaid/backend"
	"github.com/isaacev/Plaid/feedback"
	"github.com/isaacev/Plaid/frontend"
	"github.com/isaacev/Plaid/source"
	"github.com/urfave/cli"
)

var errorNoColor bool
var debugShowAST bool
var debugShowDisassembly bool
var debugShowAll bool

func readSourceFiles(args []string) (files []*source.File) {
	var filenames []string

	for _, arg := range args {
		// Try to convert every argument to an absolute path, it not possible,
		// claim the file could not be found. If a path can be produced but has
		// the wrong extension, admit defeat for that argument. If both of these
		// tests are passed, add the absolute file to the `filenames` list
		if abs, err := filepath.Abs(arg); err == nil {
			if path.Ext(abs) == ".plaid" {
				filenames = append(filenames, abs)
			} else {
				fmt.Printf("could not use '%s' with extension '%s'\n", abs, path.Ext(abs))
			}
		} else {
			fmt.Printf("could not find '%s'\n", arg)
		}
	}

	// Convert each absolute filename into a `source.File`
	for _, filename := range filenames {
		buf, err := ioutil.ReadFile(filename)

		// If any error is produced during the file read, print the error and
		// quit trying to process this filename
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		// Convert the raw byte buffer to a proper string of the file's contents
		contents := string(buf)

		// A slice of each line in the source file is cached with `source.File`
		lines := strings.SplitAfter(contents, "\n")

		// Create the `source.File` struct and append it to the list of other
		// files part of this batch
		files = append(files, &source.File{
			Filename: filename,
			Contents: contents,
			Lines:    lines,
		})
	}

	return files
}

func digestFile(file *source.File, shouldRun bool) (msgs []feedback.Message) {
	// Create a new parser and parse the file's syntax, store abstract-syntax-tree
	// in the variable `ast` and collect any errors/warnings emitted by the
	// parsing process
	var ast *frontend.ProgramNode
	ast, msgs = frontend.Parse(file)

	// Typecheck the AST and append any errors/warnings to the messages slice
	msgs = append(msgs, frontend.Check(file, ast)...)

	// Check if any of the messages are errors. If they are, stop the pipeline
	// and emit the messages. If no messages or all are warnings, continue
	for _, msg := range msgs {
		if _, ok := msg.(feedback.Error); ok {
			// At least one message is an error, stop the pipeline
			return msgs
		}
	}

	// If the `debug-ast` flag is set, output an ASCII header an an S-expression
	// AST representation
	if debugShowAll || debugShowAST {
		fmt.Println("#######################")
		fmt.Println("##        AST        ##")
		fmt.Println("#######################")
		fmt.Println()
		fmt.Println(frontend.StringifyAST(ast))
		fmt.Println()
	}

	// If the `shouldRun` parameter is false, this is as far as the function
	// needs to go since everything beyond this handles compilation and execution
	if shouldRun == false {
		return msgs
	}

	// Compile the AST into the top-level main function and all function bodies
	// defined within the file
	mainFunc, funcs := backend.Compile(ast)

	// If the `debug-disassembly` flag is set, output an ASCII header and a
	// disassembled representation all compiled functions including the main function
	if debugShowAll || debugShowDisassembly {
		fmt.Println("#######################")
		fmt.Println("##    Disassembly    ##")
		fmt.Println("#######################")
		fmt.Println()

		// Disassemble the top-level main function
		fmt.Print("main ")
		backend.Disassemble(mainFunc)
		fmt.Println()

		// Disassemble each function defined in the file
		for i, f := range funcs {
			fmt.Printf("#%d ", i)
			backend.Disassemble(f)
			fmt.Println()
		}
	}

	backend.Execute(mainFunc, funcs)
	return msgs
}

func main() {
	app := cli.NewApp()
	app.Name = "plaid"
	app.Usage = "a simple scripting language"

	noColorFlag := cli.BoolFlag{
		Name:        "no-color",
		Usage:       "hide colors in error and warning messages",
		Destination: &errorNoColor,
	}

	debugAstFlag := cli.BoolFlag{
		Name:        "debug-ast",
		Usage:       "show a basic representation of the abstract-syntax-tree",
		Destination: &debugShowAST,
	}

	debugDisFlag := cli.BoolFlag{
		Name:        "debug-disassembly",
		Usage:       "show the disassembled bytecode emitted by the compiler",
		Destination: &debugShowDisassembly,
	}

	debugAllFlag := cli.BoolFlag{
		Name:        "debug",
		Usage:       "alias for --debug-ast --debug-disassembly",
		Destination: &debugShowAll,
	}

	app.Commands = []cli.Command{
		{
			Name:    "run",
			Aliases: []string{"r"},
			Usage:   "Interpret file(s) and output any results",
			Flags: []cli.Flag{
				noColorFlag,
				debugDisFlag,
				debugAstFlag,
				debugAllFlag,
			},
			Action: func(c *cli.Context) error {
				files := readSourceFiles(c.Args())

				for _, f := range files {
					msgs := digestFile(f, true)

					if len(msgs) > 0 {
						fmt.Printf("# %s\n", f.Filename)

						for _, msg := range msgs {
							fmt.Println(msg.Make(!errorNoColor))
						}
					}
				}

				return nil
			},
		},
		{
			Name:    "check",
			Aliases: []string{"c"},
			Usage:   "Check syntax and type relationships of file(s) without executing",
			Flags: []cli.Flag{
				noColorFlag,
				debugAstFlag,
			},
			Action: func(c *cli.Context) error {
				files := readSourceFiles(c.Args())

				for _, f := range files {
					msgs := digestFile(f, false)

					if len(msgs) > 0 {
						fmt.Printf("# %s\n", f.Filename)

						for _, msg := range msgs {
							fmt.Println(msg.Make(!errorNoColor))
						}
					}
				}

				return nil
			},
		},
	}

	app.Action = func(c *cli.Context) error {
		cli.ShowAppHelp(c)
		return nil
	}

	app.Run(os.Args)
}
