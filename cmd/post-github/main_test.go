package main

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/google/go-github/github"
)

func TestRunGH(t *testing.T) {
	f, err := os.Open("testdata/fatal")
	if err != nil {
		t.Fatal(err)
	}

	const (
		expOwner = "cockroachdb"
		expRepo  = "cockroach"
		pkg      = "foo/bar/baz"
		sha      = "abcd123"
		issueID  = 1337
	)

	issueBodyRe := regexp.MustCompile(fmt.Sprintf(`(?s)\ASHA: https://github.com/cockroachdb/cockroach/commits/%s

Stress build found a failed test:

.*
F161007 00:27:33\.243126 449 storage/store\.go:2446  \[s3\] \[n3,s3,r1:/M{in-ax}\]: could not remove placeholder after preemptive snapshot
goroutine 449 \[running\]:
`, sha))

	commentBodyRe := regexp.MustCompile(`(?s)\ARun details:

.*
go test -v  -tags '' -i -c ./storage -o ./storage/stress.test
.*
I161007 00:27:32\.319758 1 rand\.go:76  Random seed: -8328855967269786437
`)

	if val, ok := os.LookupEnv(teamcityVCSNumberEnv); ok {
		defer os.Setenv(teamcityVCSNumberEnv, val)
	} else {
		defer os.Unsetenv(teamcityVCSNumberEnv)
	}

	if err := os.Setenv(teamcityVCSNumberEnv, sha); err != nil {
		t.Fatal(err)
	}

	if err := runGH(
		f,
		func(owner string, repo string, issue *github.IssueRequest) (*github.Issue, *github.Response, error) {
			if owner != expOwner {
				t.Fatalf("got %s, expected %s", owner, expOwner)
			}
			if repo != expRepo {
				t.Fatalf("got %s, expected %s", repo, expRepo)
			}
			if expected := fmt.Sprintf("%s: %s failed under stress", pkg, "TestRaftRemoveRace"); *issue.Title != expected {
				t.Fatalf("got %s, expected %s", *issue.Title, expected)
			}
			if !issueBodyRe.MatchString(*issue.Body) {
				t.Fatalf("got:\n%s\nexpected:\n%s", *issue.Body, issueBodyRe)
			}
			return &github.Issue{ID: github.Int(issueID)}, nil, nil
		},
		func(owner string, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error) {
			if owner != expOwner {
				t.Fatalf("got %s, expected %s", owner, expOwner)
			}
			if repo != expRepo {
				t.Fatalf("got %s, expected %s", repo, expRepo)
			}
			if number != issueID {
				t.Fatalf("got %d, expected %d", number, issueID)
			}
			if !commentBodyRe.MatchString(*comment.Body) {
				t.Fatalf("got:\n%s\nexpected:\n%s", *comment.Body, commentBodyRe)
			}
			return nil, nil, nil
		},
	); err != nil {
		t.Fatal(err)
	}
}
