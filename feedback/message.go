package feedback

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/isaacev/Plaid/source"
)

const (
	warningColors = iota
	errorColors   = iota
	helperColors  = iota
	noColors      = iota
)

// Message is the interface for all Warnings and Errors that can be emitted
// by the stages of the pipeline
type Message interface {
	Make(withColor bool) string
}

// Selection represents a region of the source code file along with a
// corresponding description that supplies information as to why an warning or
// error occured
type Selection struct {
	Description string
	Span        source.Span
}

// Warning classification constants
const (
	SyntaxWarning    string = "syntax warning"
	TypeCheckWarning string = "type check warning"
)

// Warning messages are emitted by the pipeline to highlight issues which might
// need to be addressed by the source code author
type Warning struct {
	Classification string
	File           *source.File
	What           Selection
	Why            []Selection
}

// Make takes a Warning and produces a fully rendered message with the option of
// using colors to make elements of the message more clear. The rendered message
// is returned as a single string and can be then output to stdout or some other
// destination
func (w Warning) Make(withColor bool) string {
	color.NoColor = !withColor
	return makeMessage(w.Classification, w.File, w.What, w.Why, warningColors)
}

// Error classification constants
const (
	SyntaxError    string = "syntax error"
	TypeCheckError string = "type check error"
)

// Error messages are more serious than warnings and typically cause the
// pipeline to be stopped. This includes illegal syntax errors, underined
// variables or type errors
type Error struct {
	Classification string
	File           *source.File
	What           Selection
	Why            []Selection
}

// Make takes an Error and produces a fully rendered message with the option of
// using colors to make elements of the message more clear. The rendered message
// is returned as a single string and can be then output to stdout or some other
// destination
func (e Error) Make(withColor bool) string {
	color.NoColor = !withColor
	return makeMessage(e.Classification, e.File, e.What, e.Why, errorColors)
}

// makeMessage is a utility function which takes any Message and a corresponding
// File to make a rendered message of the form:
//
// <message type>: <error classification>
//   --> <filename>:<line number>:<column number>
//    |
//  1 | <offending line of source code>
//    |  ^^^^^^^^^ <message detailing error>
//
func makeMessage(classification string, file *source.File, what Selection, why []Selection, colorScheme int) string {
	yellowBold := color.New(color.FgYellow, color.Bold).SprintFunc()
	redBold := color.New(color.FgRed, color.Bold).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	var lines []string
	var header string
	var placeValues int

	maxLineNum := getMaxLineNum(append([]Selection{what}, why...)...)
	placeValues = utf8.RuneCountInString(fmt.Sprintf("%d", maxLineNum))

	if colorScheme == warningColors {
		header = "warning:"
		lines = append(lines, yellowBold(fmt.Sprintf("%s %s", header, classification)))
	} else {
		header = "error:"
		lines = append(lines, redBold(fmt.Sprintf("%s %s", header, classification)))
	}

	lines = append(lines, fmt.Sprintf(" %s%s %s:%d:%d",
		mulStr(" ", placeValues),
		blue("-->"),
		file.Filename,
		what.Span.Start.Line,
		what.Span.Start.Col))

	lines = append(lines, blue(fmt.Sprintf(" %s |", mulStr(" ", placeValues))))

	for i, sel := range why {
		if i > 0 && why[i-1].Span.End.Line < sel.Span.Start.Line {
			lines = append(lines, blue("..."))
		}

		lines = append(lines, sourceCodeSelection(file, sel, helperColors, placeValues)...)
	}

	if len(why) > 0 {
		prevLastLine := why[len(why)-1].Span.End.Line
		startLine := what.Span.Start.Line

		if prevLastLine+1 < startLine {
			lines = append(lines, fmt.Sprintf(" %s%s", mulStr(" ", placeValues), blue("...")))
		} else {
			for i := prevLastLine + 1; i < startLine; i++ {
				lines = append(lines, sourceCodeSelection(file, Selection{
					Span: source.Span{
						Start: source.Pos{Line: i, Col: 1},
						End:   source.Pos{Line: i, Col: utf8.RuneCountInString(file.Lines[i]) - 1},
					},
				}, noColors, placeValues)...)
			}
		}
	}

	lines = append(lines, sourceCodeSelection(file, what, colorScheme, placeValues)...)
	return strings.Join(lines, "\n")
}

// sourceCodeSelection is a utility function which, given a File and a Selection
// extracts an offending line of source code from the source file and renders
// the line along with its line number and the description set to accompany that
// line of source code
func sourceCodeSelection(file *source.File, sel Selection, colorScheme int, placeValues int) (lines []string) {
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	sourceLines := file.Lines[sel.Span.Start.Line-1 : sel.Span.End.Line]

	// In order to maintain the alignment of all source code selections no
	// matter the number of numerals in their respective line numbers, each
	// selection's margin is dynamically shifted to the widest margin needed by
	// the largest line number in ANY included selection. The following lines
	// create template strings with dynamically chosen left padding to be used
	// when formatting the selection
	numMargFmt := fmt.Sprintf("%%%dd", placeValues)
	emptyMargFmt := mulStr(" ", placeValues)

	for i, srcLine := range sourceLines {
		lineNum := sel.Span.Start.Line + i
		lineNumFmt := fmt.Sprintf(numMargFmt, lineNum)

		// Replace any newline characters in a string with a plain space
		srcLine = strings.Replace(srcLine, "\n", " ", -1)

		var focusStart, focusEnd int

		if lineNum == sel.Span.Start.Line {
			focusStart = sel.Span.Start.Col
		} else {
			focusStart = 1
		}

		if lineNum == sel.Span.End.Line {
			focusEnd = sel.Span.End.Col + 1
		} else {
			focusEnd = utf8.RuneCountInString(srcLine)
		}

		prefix, focus, suffix := highlightSourceLine(srcLine, focusStart, focusEnd, colorScheme)

		if colorScheme == warningColors {
			focus = yellow(focus)
		} else if colorScheme == errorColors {
			focus = red(focus)
		} else if colorScheme == helperColors {
			focus = blue(focus)
		}

		lines = append(lines, fmt.Sprintf(" %s %s %s%s%s", blue(lineNumFmt), blue("|"), prefix, focus, suffix))
	}

	if sel.Description == "" {
		return lines
	}

	var underlineChar string
	var desc string

	if colorScheme == warningColors {
		underlineChar = yellow("^")
		desc = yellow(sel.Description)
	} else if colorScheme == errorColors {
		underlineChar = red("^")
		desc = red(sel.Description)
	} else {
		underlineChar = blue("-")
		desc = blue(sel.Description)
	}

	leftPad := mulStr(" ", sel.Span.Start.Col-1)

	// Underline width must be at least 1 character wide
	underline := mulStr(underlineChar, int(math.Max(float64((sel.Span.End.Col+1)-sel.Span.Start.Col), 1)))
	lines = append(lines, fmt.Sprintf(" %s %s %s%s %s", emptyMargFmt, blue("|"), leftPad, underline, desc))

	return lines
}

// getMaxLineNum returns the largest line number present in a collection of
// Selection structs
func getMaxLineNum(selections ...Selection) (max int) {
	max = 1

	for _, sel := range selections {
		if sel.Span.End.Line > max {
			max = sel.Span.End.Line
		}
	}

	return max
}

// highlightSourceLine takes a line of source code and 2 column numbers and
// returns the segment before the first column number, the segment between the
// column numbers, and the segment after the last column number. This is used
// to provide color to only the significant segment of a source code line
func highlightSourceLine(line string, start, end, colorScheme int) (prefix, focus, suffix string) {
	nextByte := 0

	for i := 1; i < end; i++ {
		runeValue, runeWidth := utf8.DecodeRuneInString(line[nextByte:])
		nextByte += runeWidth

		if i < start {
			prefix += string(runeValue)
		} else {
			focus += string(runeValue)
		}
	}

	suffix = line[nextByte:]

	return prefix, focus, suffix
}

// mulStr repeats a string "n" times
func mulStr(s string, n int) (out string) {
	for ; n > 0; n-- {
		out += s
	}

	return out
}
