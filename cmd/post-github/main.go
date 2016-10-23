package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/tebeka/go2xunit/lib"
)

const githubAPITokenEnv = "GITHUB_API_TOKEN"
const teamcityVCSNumberEnv = "BUILD_VCS_NUMBER"

func main() {
	token, ok := os.LookupEnv(githubAPITokenEnv)
	if !ok {
		log.Fatalf("GitHub API token environment variable %s is not set", githubAPITokenEnv)
	}

	client := github.NewClient(oauth2.NewClient(oauth2.NoContext, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)))

	if err := runGH(os.Stdin, client.Issues.Create, client.Issues.CreateComment); err != nil {
		log.Fatal(err)
	}
}

func runGH(
	input io.Reader,
	createIssue func(owner string, repo string, issue *github.IssueRequest) (*github.Issue, *github.Response, error),
	createComment func(owner string, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error),
) error {
	sha, ok := os.LookupEnv(teamcityVCSNumberEnv)
	if !ok {
		return errors.Errorf("VCS number environment variable %s is not set", teamcityVCSNumberEnv)
	}

	suites, err := lib.ParseGotest(input, "")
	if err != nil {
		return errors.Wrap(err, "failed to parse `go test` output")
	}
	for _, suite := range suites {
		for _, test := range suite.Tests {
			switch test.Status {
			case lib.Failed:
				title := fmt.Sprintf("%s: %s failed under stress", suite.Name, test.Name)
				body := fmt.Sprintf(`SHA: https://github.com/cockroachdb/cockroach/commits/%s

Stress build found a failed test:

%s`, sha, "```\n"+test.Message+"\n```")

				issueRequest := &github.IssueRequest{
					Title: &title,
					Body:  &body,
					Labels: &[]string{
						"Robot",
						"test-failure",
					},
				}
				if _, _, err := createIssue("cockroachdb", "cockroach", issueRequest); err != nil {
					return errors.Wrapf(err, "failed to create GitHub issue %s", github.Stringify(issueRequest))
				}

			}
		}
	}

	return nil
}
