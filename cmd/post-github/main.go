package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"

	"github.com/cockroachdb/build-utils/parser"
)

const githubAPITokenEnv = "GITHUB_API_TOKEN"
const importPathUnderTestEnv = "PKG"
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

func makeCodeLike(output []string) string {
	const codeDelim = "```"
	output = append(output, codeDelim)
	output = append([]string{codeDelim}, output...)
	return strings.Join(output, "\n")
}

func runGH(
	input io.Reader,
	createIssue func(owner string, repo string, issue *github.IssueRequest) (*github.Issue, *github.Response, error),
	createComment func(owner string, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error),
) error {

	importPath, ok := os.LookupEnv(importPathUnderTestEnv)
	if !ok {
		return errors.Errorf("import path environment variable %s is not set", importPathUnderTestEnv)
	}

	sha, ok := os.LookupEnv(teamcityVCSNumberEnv)
	if !ok {
		return errors.Errorf("VCS number environment variable %s is not set", teamcityVCSNumberEnv)
	}

	var issues []*github.Issue
	return parser.ForEachTest(input, func(test parser.Test, final bool) error {
		if final {
			for _, issue := range issues {
				body := fmt.Sprintf(`Run details:

%s`, makeCodeLike(test.Output))
				if _, _, err := createComment("cockroachdb", "cockroach", *issue.ID, &github.IssueComment{
					Body: &body,
				}); err != nil {
					return errors.Wrapf(err, "failed to post run details on GitHub issue %s", github.Stringify(issue))
				}
			}

			return nil
		}

		switch {
		case test.Pass, test.Skip:
			return nil
		}
		title := fmt.Sprintf("%s: %s failed under stress", importPath, test.Name)
		body := fmt.Sprintf(`SHA: https://github.com/cockroachdb/cockroach/commits/%s

Stress build found a failed test:

%s`, sha, makeCodeLike(test.Output))

		issueRequest := &github.IssueRequest{
			Title: &title,
			Body:  &body,
			Labels: &[]string{
				"Robot",
				"test-failure",
			},
		}
		issue, _, err := createIssue("cockroachdb", "cockroach", issueRequest)
		if err != nil {
			return errors.Wrapf(err, "failed to create GitHub issue %s", github.Stringify(issueRequest))
		}

		issues = append(issues, issue)
		return nil
	})
}
