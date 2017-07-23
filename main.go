package main

import (
	"flag"
	"strconv"

	"fmt"

	"github.com/fatih/color"
	"github.com/jeffijoe/total-rename/casing"
	"github.com/jeffijoe/total-rename/cli"
	"github.com/jeffijoe/total-rename/lister"
	"github.com/jeffijoe/total-rename/scanner"
	"github.com/jeffijoe/total-rename/util"
)

func main() {
	help := flag.Bool("help", false, "Shows the help menu")
	dryRun := flag.Bool("dry", false, "If set, won't rename anything.")
	force := flag.Bool("force", false, "Replaces all occurences without asking")
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	if *dryRun {
		fmt.Println("Dry run active, won't rename anything.")
	}

	if *force {
		fmt.Println("Not gonna ask for permission")
	}
	if flag.NArg() < 3 {
		fmt.Println("Not enough arguments, expects 3: <path> <needle> <replacement>")
		return
	}
	promptOccurences(flag.Arg(0), flag.Arg(1), flag.Arg(2))
	//tm.Flush()
}

func promptOccurences(path, needle, replacement string) (scanner.OccurenceGroups, error) {
	nodes, err := lister.ListFileNodes(util.GetWD(), path)
	if err != nil {
		return nil, err
	}
	groups, err := scanner.ScanFileNodes(nodes, needle)
	if err != nil {
		return nil, err
	}
	replacementVariants := casing.GenerateCasings(replacement)
	result := scanner.OccurenceGroups{}
	for _, group := range groups {
		var newGroup *scanner.OccurenceGroup
		switch group.Type {
		case scanner.OccurenceGroupTypeContent:
			newGroup, err = promptContentGroup(group, replacementVariants)
			// case scanner.OccurenceGroupTypePath:
			// 	newGroup, err = promptPathGroup(group, replacementVariants)
		}
		if err != nil {
			return nil, err
		}
		if newGroup != nil {
			result = append(result, newGroup)
		}
	}

	return scanner.OccurenceGroups{}, nil
}

func promptContentGroup(group *scanner.OccurenceGroup, replacementVariants casing.Variants) (*scanner.OccurenceGroup, error) {
	w := cli.Clearable()

	occurences := scanner.Occurences{}
	result := &scanner.OccurenceGroup{
		Path:       group.Path,
		Type:       group.Type,
		Occurences: occurences,
	}
	countReplaced := 0
	countSkipped := 0
	printFileStatus := func(printf func(string, ...interface{}) (int, error)) {
		color.Set(color.BgWhite)
		color.Set(color.FgBlack)
		printf(group.Path)
		color.Set(color.BgGreen)

		if countReplaced > 0 {
			printf(" %d replaced", countReplaced)
		}
		if countSkipped > 0 {
			if countReplaced > 0 {
				printf(", %d skipped", countSkipped)
			} else {
				printf(": %d skipped", countSkipped)

			}
		}
		color.Unset()
		printf("\n")
	}
	for _, oc := range group.Occurences {
		printFileStatus(w.Printf)
		w.Println()
		shouldReplace, err := promptContentOccurence(oc, replacementVariants, w)
		if err != nil {
			return nil, err
		}
		if shouldReplace {
			countReplaced = countReplaced + 1
			occurences = append(occurences, oc)
		} else {
			countSkipped = countSkipped + 1
		}

		w.Clear()
	}
	printFileStatus(fmt.Printf)
	if len(occurences) == 0 {
		return nil, nil
	}
	return result, nil
}

func promptContentOccurence(occurence *scanner.Occurence, replacementVariants casing.Variants, w *cli.Wrapper) (bool, error) {
	color.Set(color.FgHiBlack)
	for i, ln := range occurence.SurroundingLinesBefore {
		lineNum := occurence.LineNumber + i + 1 - len(occurence.SurroundingLinesBefore)
		w.Println(formatLine(lineNum, ln))
	}
	beforeMatch := occurence.Line[:occurence.LineStartIndex]
	afterMatch := occurence.Line[occurence.LineStartIndex+len(occurence.Match):]
	w.Printf(formatLine(occurence.LineNumber+1, beforeMatch))
	color.Set(color.FgYellow)
	w.Print(occurence.Match)
	color.Set(color.FgHiBlack)
	w.Println(afterMatch)
	for i, ln := range occurence.SurroundingLinesAfter {
		lineNum := occurence.LineNumber + i + 2
		w.Println(formatLine(lineNum, ln))
	}
	color.Unset()
	w.Println()
	color.Set(color.FgWhite)
	w.Print("Replace ")
	color.Set(color.FgYellow)
	w.Print(occurence.Match)
	color.Set(color.FgWhite)
	w.Print(" with ")
	color.Set(color.FgGreen)
	w.Print(replacementVariants.GetVariant(occurence.Casing).Value)
	color.Set(color.FgWhite)
	w.Println("? [Y/n] ")
	response, err := w.Confirm(true)
	return response, err
}

func formatLine(lineNum int, str string) string {
	return fmt.Sprintf("%6s: %s", strconv.Itoa(lineNum), str)
}

func printHelp() {
	fmt.Println("total-rename - case-preserving renaming utility")
	fmt.Println("")
	fmt.Println("OPTIONS:")
	fmt.Println("    --dry     If set, won't rename anything")
	fmt.Println("    --force   Replaces all occurences without asking")
	fmt.Println("    --help    Shows this help text")
	fmt.Println("")
	fmt.Println("EXAMPLE:")
	fmt.Println("")
	fmt.Println("    total-rename \"**/*.txt\" \"awesome\" \"excellent\"")
	fmt.Println("")
	fmt.Println("    Rename all occurences of \"awesome\" to \"excellent\" in")
	fmt.Println("    all .txt files (and folders) recursively from the")
	fmt.Println("    current working directory:")
	fmt.Println("")
	fmt.Println("EXAMPLE:")
	fmt.Println("")
	fmt.Println("    total-rename \"/Users/jeff/projects/my-app/src/**/*.*\" \"awesome\" \"excellent\"")
	fmt.Println("")
	fmt.Println("    Like the first example, but from an absolute path, and match")
	fmt.Println("    all file extensions.")
	fmt.Println("")
}
