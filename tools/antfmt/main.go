// Command antfmt spreads out what gofmt and golines leave condensed: a literal
// with many keyed fields, and a wrapped list with several items on one line. They
// only look at line length, so a short line of fields gets past them.
//
//	antfmt [-l] path...
//
// It also opens up a one-line function on a line that is too long, and puts a blank
// line around statements that span lines, and above an if, for, or switch that does
// not check what the line above it made.
//
// It rewrites the .go files under each path, or with -l only lists them and exits
// with status 1, for use in a lint step. testdata, vendor, and hidden folders are
// skipped.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	list := flag.Bool("l", false, "list files that need formatting, and change nothing")

	flag.Parse()

	changed := 0

	for _, root := range flag.Args() {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return err
			case entry.IsDir() && skipped(path, root):
				return filepath.SkipDir
			case entry.IsDir() || !strings.HasSuffix(path, ".go"):
				return nil
			}

			differs, err := process(path, *list)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}

			if differs {
				changed++
			}

			return nil
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}

	if *list && changed > 0 {
		os.Exit(1)
	}
}

func skipped(path, root string) bool {
	name := filepath.Base(path)

	return path != root && (name == "testdata" || name == "vendor" || strings.HasPrefix(name, "."))
}

func process(path string, list bool) (bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	out, err := Format(src)
	if err != nil {
		return false, err
	}

	if bytes.Equal(src, out) {
		return false, nil
	}

	if list {
		fmt.Println(path)

		return true, nil
	}

	return true, os.WriteFile(path, out, 0o644)
}
