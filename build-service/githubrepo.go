package main

import (
	"fmt"
	"regexp"
	"time"

	"github.com/andreas/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/robfig/goauth2/oauth"
)

var (
	githubImportPathRe = regexp.MustCompile(`^github\.com/([a-zA-Z0-9-_]+)/([a-zA-Z0-9-_]+)$`)
)

type GithubRepo struct {
	client *github.Client
}

func NewGithubRepo(accessToken string) *GithubRepo {
	cacheT := httpcache.NewMemoryCacheTransport()
	oauthT := &oauth.Transport{
		Transport: cacheT,
		Token:     &oauth.Token{AccessToken: accessToken},
	}

	return &GithubRepo{
		client: github.NewClient(oauthT.Client()),
	}
}

func (r *GithubRepo) MergeBaseDate(importPath, sha, base string) (*time.Time, error) {
	match := githubImportPathRe.FindStringSubmatch(importPath)
	if match == nil {
		return nil, fmt.Errorf("Import path is not a github repo")
	}

	compare, _, err := r.client.Repositories.CompareCommits(match[1], match[2], sha, base)
	if err != nil {
		return nil, err
	}

	return compare.MergeBaseCommit.Commit.Committer.Date, nil
}
