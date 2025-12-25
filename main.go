package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

const (
	scannerMaxCapacity = 10 * 1024 * 1024
)

type options struct {
	ignoreCase  bool
	lineNumber  bool
	invertMatch bool
	recursive   bool
}

func main() {
	opts := options{}
	flag.BoolVar(&opts.ignoreCase, "i", false, "ignore case distinctions")
	flag.BoolVar(&opts.lineNumber, "n", false, "print line numbers")
	flag.BoolVar(&opts.invertMatch, "v", false, "select non-matching lines")
	flag.BoolVar(&opts.recursive, "r", false, "recursively search directories")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] PATTERN [FILE ...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		exitWithError(errors.New("pattern is required"))
	}

	pattern := args[0]
	fileArgs := args[1:]

	if opts.ignoreCase {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		exitWithError(fmt.Errorf("invalid pattern: %w", err))
	}

	foundAny := false
	hadError := false

	matcher := func(line string) bool {
		matched := re.MatchString(line)
		if opts.invertMatch {
			return !matched
		}
		return matched
	}

	printer := func(name string, lineNumber int, line string, prefixName bool) {
		if prefixName {
			fmt.Printf("%s:", name)
		}
		if opts.lineNumber {
			fmt.Printf("%d:", lineNumber)
		}
		fmt.Println(line)
	}

	if len(fileArgs) == 0 {
		matched, err := grepReader(os.Stdin, "(stdin)", matcher, printer, opts, false)
		if err != nil {
			hadError = true
			fmt.Fprintln(os.Stderr, err)
		}
		if matched {
			foundAny = true
		}
	} else {
		multipleInputs := len(fileArgs) > 1
		for _, path := range fileArgs {
			info, err := os.Stat(path)
			if err != nil {
				hadError = true
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			if info.IsDir() {
				if !opts.recursive {
					hadError = true
					fmt.Fprintf(os.Stderr, "%s: is a directory (use -r)\n", path)
					continue
				}
				err = filepath.WalkDir(path, func(walkPath string, d os.DirEntry, walkErr error) error {
					if walkErr != nil {
						hadError = true
						fmt.Fprintln(os.Stderr, walkErr)
						return nil
					}
					if d.IsDir() {
						return nil
					}
					matched, err := grepFile(walkPath, matcher, printer, opts, true)
					if err != nil {
						hadError = true
						fmt.Fprintln(os.Stderr, err)
						return nil
					}
					if matched {
						foundAny = true
					}
					return nil
				})
				if err != nil {
					hadError = true
					fmt.Fprintln(os.Stderr, err)
				}
				continue
			}

			matched, err := grepFile(path, matcher, printer, opts, multipleInputs)
			if err != nil {
				hadError = true
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			if matched {
				foundAny = true
			}
		}
	}

	if hadError {
		os.Exit(2)
	}
	if foundAny {
		os.Exit(0)
	}
	os.Exit(1)
}

func grepFile(path string, matcher func(string) bool, printer func(string, int, string, bool), opts options, prefixName bool) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	return grepReader(file, path, matcher, printer, opts, prefixName)
}

func grepReader(reader io.Reader, name string, matcher func(string) bool, printer func(string, int, string, bool), opts options, prefixName bool) (bool, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerMaxCapacity)

	matchedAny := false
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if matcher(line) {
			matchedAny = true
			printer(name, lineNumber, line, prefixName)
		}
	}

	if err := scanner.Err(); err != nil {
		return matchedAny, err
	}

	return matchedAny, nil
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
