// Package comment upserts ephemeractl's single sticky cost comment on a PR.
package comment

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-github/v66/github"
)

// Poster manages the sticky comment for one PR.
type Poster struct {
	gh     *github.Client
	owner  string
	repo   string
	number int
}

// NewPoster builds a Poster authenticated with token for owner/repo#number.
func NewPoster(token, owner, repo string, number int) *Poster {
	return &Poster{
		gh:     github.NewClient(nil).WithAuthToken(token),
		owner:  owner,
		repo:   repo,
		number: number,
	}
}

// setBaseURL points the client at an alternate API root (used in tests).
func (p *Poster) setBaseURL(base string) error {
	u, err := url.Parse(base)
	if err != nil {
		return err
	}
	p.gh.BaseURL = u
	return nil
}

// Upsert finds the comment whose body contains marker and edits it; if none
// exists it creates one. It returns the comment's HTML URL.
func (p *Poster) Upsert(ctx context.Context, marker, body string) (string, error) {
	opt := &github.IssueListCommentsOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for {
		comments, resp, err := p.gh.Issues.ListComments(ctx, p.owner, p.repo, p.number, opt)
		if err != nil {
			return "", fmt.Errorf("list PR comments: %w", err)
		}
		for _, c := range comments {
			if c.GetBody() != "" && strings.Contains(c.GetBody(), marker) {
				edited, _, err := p.gh.Issues.EditComment(ctx, p.owner, p.repo, c.GetID(),
					&github.IssueComment{Body: github.String(body)})
				if err != nil {
					return "", fmt.Errorf("edit sticky comment: %w", err)
				}
				return edited.GetHTMLURL(), nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	created, _, err := p.gh.Issues.CreateComment(ctx, p.owner, p.repo, p.number,
		&github.IssueComment{Body: github.String(body)})
	if err != nil {
		return "", fmt.Errorf("create sticky comment: %w", err)
	}
	return created.GetHTMLURL(), nil
}
