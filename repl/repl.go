package repl

import (
	"bufio"
	"fmt"
	"io"

	"github.com/titivuk/go-interpreter/evaluator"
	"github.com/titivuk/go-interpreter/lexer"
	"github.com/titivuk/go-interpreter/object"
	"github.com/titivuk/go-interpreter/parser"
)

const PROMT = ">> "
const MONKEY_FACE = `            __,__
   .--.  .-"     "-.  .--.
  / .. \/  .-. .-.  \/ .. \
 | |  '|  /   Y   \  |'  | |
 | \   \  \ 0 | 0 /  /   / |
  \ '- ,\.-"""""""-./, -' /
   ''-' /_   ^ ^   _\ '-''
       |  \._   _./  |
       \   \ '~' /   /
        '._ '-=-' _.'
           '-----'
`

func Start(in io.Reader, out io.Writer) {
	// Scanner provides a convenient interface for reading data such as a file of newline-delimited lines of text.
	// Successive calls to the Scan method will step through the 'tokens' of a file, skipping the bytes between the tokens.
	// The specification of a token is defined by a split function of type SplitFunc; the default split function breaks the input into lines with line termination stripped.
	// Split functions are defined in this package for scanning a file into lines, bytes, UTF-8-encoded runes, and space-delimited words.
	// The client may instead provide a custom split function.
	scanner := bufio.NewScanner(in)
	env := object.NewEnvironment()

	for {
		fmt.Fprintf(out, PROMT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		// Text returns the most recent token generated by a call to Scan as a newly allocated string holding its bytes.
		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		evaluated := evaluator.Eval(program, env)
		if evaluated != nil {
			io.WriteString(out, evaluated.Inspect())
			io.WriteString(out, "\n")
		}
	}

}

func printParserErrors(out io.Writer, errors []string) {
	io.WriteString(out, MONKEY_FACE)
	io.WriteString(out, "Woops! We ran into some monkey business here!\n")
	io.WriteString(out, " parser errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
