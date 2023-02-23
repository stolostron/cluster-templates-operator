package repository

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	githttpclient "github.com/go-git/go-git/v5/plumbing/transport/client"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

type GitRepositoryIndex struct {
	Branches []string `json:"branches,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Error    string   `json:"error,omitempty"`
	Url      string   `json:"url,omitempty"`
	Name     string   `json:"name,omitempty"`
}

func GetGitInfo(customClient *HttpClient, repoUrl string) ([]string, []string, error) {
	// Setup the remote from url:
	listOptions := &git.ListOptions{}
	rem := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoUrl},
	})
	githttpclient.InstallProtocol("https", githttp.NewClient(customClient.client))

	// Set baisc auth method for git remote list:
	secret := customClient.secret
	if secret != nil {
		var username []byte
		var password []byte
		if usernameSecret, usernameOk := secret.Data[RepoSecretUsername]; usernameOk {
			username = usernameSecret
		}
		if passwordSecret, passwordOk := secret.Data[RepoSecretPassword]; passwordOk {
			password = passwordSecret
		}
		if username != nil && string(username) != "" && password != nil {
			listOptions.Auth = &githttp.BasicAuth{
				Username: string(username),
				Password: string(password),
			}
		}
		// Token auth is used if username is empty and password is set:
		if (username == nil || string(username) == "") && password != nil {
			listOptions.Auth = &githttp.TokenAuth{
				Token: string(password),
			}
		}
	}

	// Execetue git remote-ls:
	refs, err := rem.List(listOptions)
	if err != nil {
		return nil, nil, err
	}

	// Parse the remote data:
	var tags []string
	var branches []string
	for _, ref := range refs {
		if ref.Name().IsTag() {
			tags = append(tags, ref.Name().Short())
		}
		if ref.Name().IsBranch() {
			branches = append(branches, ref.Name().Short())
		}
	}

	return tags, branches, nil
}
