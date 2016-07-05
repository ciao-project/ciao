//
// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"os"
	"strings"
	"time"
)

const repoOwner = "01org"
const repo = "ciao"

var rcSHA = flag.String("head", "", "The sha of the release candidate")

func main() {
	flag.Parse()

	f, err := os.Create("release.txt")
	if err != nil {
		glog.Error(err)
		os.Exit(1)
	}
	defer f.Close()

	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken == "" {
		glog.Fatal("You must set GITHUB_TOKEN env var")
		os.Exit(1)
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)

	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	headCommit, _, _ := client.Repositories.GetCommit(repoOwner, repo, *rcSHA)
	if headCommit == nil {
		glog.Fatal("The SHA was not valid")
		os.Exit(1)
	}

	release, _, _ := client.Repositories.GetLatestRelease(repoOwner, repo)

	var tags []*github.RepositoryTag
	var tagsOpts github.ListOptions

	for {
		t, resp, err := client.Repositories.ListTags(repoOwner, repo, &tagsOpts)
		if err != nil {
			glog.Fatal(err)
		}

		tags = append(tags, t...)

		if resp.NextPage == 0 {
			break
		}

		tagsOpts.Page = resp.NextPage
	}

	var lastRelease time.Time

	for _, t := range tags {
		if *t.Name == *release.TagName {
			c, _, _ := client.Repositories.GetCommit(repoOwner, repo, *t.Commit.SHA)
			if c == nil {
				glog.Fatal("The SHA was not valid")
				os.Exit(1)
			}
			lastRelease = *c.Commit.Committer.Date
		}
	}

	rc := *headCommit.Commit.Committer.Date

	fmt.Fprintf(f, "Changes since last release\n\n")

	var prs []*github.PullRequest

	prOpts := github.PullRequestListOptions{
		State: "closed",
		Base:  "master",
	}

	for {
		pr, resp, err := client.PullRequests.List(repoOwner, repo, &prOpts)
		if err != nil {
			glog.Fatal(err)
		}

		prs = append(prs, pr...)

		if resp.NextPage == 0 {
			break
		}

		prOpts.Page = resp.NextPage
	}

	prmap := make(map[int]github.PullRequest)
	for _, pr := range prs {
		prmap[*pr.Number] = *pr
	}

	var events []*github.IssueEvent
	var eventOpts github.ListOptions

	for {
		e, resp, err := client.Issues.ListRepositoryEvents(repoOwner, repo, &eventOpts)
		if err != nil {
			glog.Fatal(err)
		}

		events = append(events, e...)

		if resp.NextPage == 0 {
			break
		}

		eventOpts.Page = resp.NextPage
	}

	eventsmap := make(map[string][]*github.IssueEvent)

	for _, e := range events {
		key := *e.Event

		if key == "merged" || key == "closed" {
			num := *e.Issue.Number

			if e.Issue.PullRequestLinks != nil {
				_, ok := prmap[num]
				if !ok {
					continue
				}
			}

			if lastRelease.IsZero() {
				eventsmap[key] = append(eventsmap[key], e)
			} else {
				if lastRelease.Before(*e.CreatedAt) {
					if e.CreatedAt.Before(rc) {
						eventsmap[key] = append(eventsmap[key], e)
					}
				}
			}
		}
	}

	for key, list := range eventsmap {
		fmt.Fprintf(f, "---%s---\n", key)
		for _, e := range list {
			i := *e.Issue
			fmt.Fprintf(f, "\tIssue/PR #%d: %s\n", *i.Number, *i.Title)
			fmt.Fprintf(f, "\tURL: %s\n\n", *i.HTMLURL)
		}
		fmt.Fprintf(f, "\n")
	}

	copts := github.CommitsListOptions{
		Since: lastRelease,
		Until: rc,
	}

	var commits []*github.RepositoryCommit
	for {
		c, resp, err := client.Repositories.ListCommits(repoOwner, repo, &copts)
		if err != nil {
			glog.Fatal(err)
		}

		commits = append(commits, c...)
		if resp.NextPage == 0 {
			break
		}
		if resp != nil {
			copts.ListOptions.Page = resp.NextPage
		}
	}

	if len(commits) > 0 {
		fmt.Fprintln(f, "---Full Change Log---")
	}

	for _, c := range commits {
		lines := strings.Split(*c.Commit.Message, "\n")
		fmt.Fprintf(f, "\t%s\n", lines[0])
	}
}
