package parser

import (
	"bufio"
	"io"
	"regexp"

	"github.com/pkg/errors"
)

var (
	run   = regexp.MustCompile("=== RUN\\s+([a-zA-Z_]\\S*)")
	end   = regexp.MustCompile("--- (PASS|SKIP|FAIL):\\s+([a-zA-Z_]\\S*) \\(([\\.\\d]+)")
	suite = regexp.MustCompile("^(ok|FAIL)\\s+([^\\s]+)\\s+([\\.\\d]+)s")
	race  = regexp.MustCompile("^WARNING: DATA RACE")
)

// Test models a single Go test execution and its outcome.
type Test struct {
	Name                   string
	Output                 []string
	Race, Fail, Skip, Pass bool
}

// ForEachTest parses a `go test` run from reader and invokes outputTest with
// each test result.
func ForEachTest(reader io.Reader, outputFn func(_ Test, final bool) error) error {
	scanner := bufio.NewScanner(reader)

	return mungeTest(Test{}, scanner, outputFn)
}

func mungeTest(test Test, scanner *bufio.Scanner, outputFn func(_ Test, final bool) error) error {
	for scanner.Scan() {
		line := scanner.Text()
		if runOut := run.FindStringSubmatch(line); runOut != nil {
			if err := mungeTest(Test{Name: runOut[1]}, scanner, outputFn); err != nil {
				return err
			}
			continue
		}

		if endOut := end.FindStringSubmatch(line); endOut != nil {
			switch endOut[1] {
			case "FAIL":
				test.Fail = true
			case "SKIP":
				test.Skip = true
			case "PASS":
				test.Pass = true
			}
			if testName := endOut[2]; testName != test.Name {
				return errors.Errorf("expected to find end of %s, found end of %s", test.Name, testName)
			}
			return outputFn(test, false)
		}

		if race.MatchString(line) {
			test.Race = true
			return outputFn(test, false)
		}

		if suiteOut := suite.FindStringSubmatch(line); suiteOut != nil {
			test.Fail = true
			return outputFn(test, false)
		}

		test.Output = append(test.Output, line)
	}

	if err := outputFn(test, len(test.Name) == 0); err != nil {
		return err
	}

	return scanner.Err()
}
