// Copyright 2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not
// use this file except in compliance with the License. A copy of the
// License is located at
//
// http://aws.amazon.com/apache2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

// Package githubclient contains methods for interacting with git
package githubclient

import (
	"github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/go-github/github"
	gitcontext "golang.org/x/net/context"

	"fmt"
	"net/http"
	"strings"
)

const (
	defaultBranch = "master"
)

// NewClient is a constructor for GitClient
func NewClient(httpClient *http.Client) IGitClient {

	return &GitClient{
		github.NewClient(httpClient),
	}
}

// GitClient is a wrapper around github.Client. This is done for mocking
type GitClient struct {
	*github.Client
}

// IGitClient is an interface for type IGitClient
type IGitClient interface {
	GetRepositoryContents(log log.T, owner, repo, path string, opt *github.RepositoryContentGetOptions) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, err error)
	ParseGetOptions(log log.T, getOptions string) (*github.RepositoryContentGetOptions, error)
}

// GetRepositoryContents is a wrapper around GetContents method in gitub SDK
func (git *GitClient) GetRepositoryContents(log log.T, owner, repo, path string, opt *github.RepositoryContentGetOptions) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, err error) {
	var resp *github.Response

	fileContent, directoryContent, resp, err = git.Repositories.GetContents(gitcontext.Background(), owner, repo, path, opt)

	if fileContent != nil {
		log.Info("URL downloaded from - ", fileContent.GetURL())
	}

	defer resp.Body.Close()
	log.Info("Status code - ", resp.StatusCode)
	// TODO: meloniam@ Check github documentation and add checks for other status codes.
	if err != nil {

		log.Error("Error retreiving information from github repository. Error - %v and response - %v", err, resp)
		return nil, nil, err
	} else if resp.StatusCode == http.StatusForbidden && resp.Rate.Limit == 0 {
		return nil, nil, fmt.Errorf("Rate limit exceeded")

	} else if resp.StatusCode == http.StatusNotFound {
		return nil, nil, fmt.Errorf("Response is - %v", resp.Status)

	}

	return fileContent, directoryContent, err
}

// ParseGetOptions manipulates the getOptions parameter and returns
func (git *GitClient) ParseGetOptions(log log.T, getOptions string) (*github.RepositoryContentGetOptions, error) {
	//If no option is specified, use master branch
	if getOptions == "" {
		return &github.RepositoryContentGetOptions{
			Ref: defaultBranch,
		}, nil
	}

	// Checking for format of extra option specified (if it has been)
	// Ideal input pattern will either be "branch: <name of branch>" or "commitID: <SHA of commit>"
	// Only one among the above patterns is valid.
	log.Debug("Splitting getOptions to get the actual option - ", getOptions)
	// TODO: meloniam@ Will a regex be more reliable here?
	branchOrSHA := strings.Split(getOptions, ":")
	if len(branchOrSHA) == 2 {
		if strings.Compare(branchOrSHA[0], "branch") != 0 && strings.Compare(branchOrSHA[0], "commitID") != 0 {
			return nil, fmt.Errorf("Type of option is unknown. Please use either 'branch' or 'commitID'.")
		}
		//Error if extra option has been specified but is empty
		// Length must be 2 (key and value)
		if branchOrSHA[1] == "" {
			return nil, fmt.Errorf("Option for retreiving git content is empty")
		}
	} else if len(branchOrSHA) > 2 {
		return nil, fmt.Errorf("Only specify one required option")
	} else {
		return nil, fmt.Errorf("getOptions is not specified in the right format")
	}
	log.Info("GetOptions value - ", branchOrSHA[1])

	return &github.RepositoryContentGetOptions{
		Ref: branchOrSHA[1],
	}, nil
}
