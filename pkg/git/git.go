package git

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

type GitClient struct {
	repo *git.Repository
	auth *ssh.PublicKeys
}

func NewClient(sshURL, sshKey string) (*GitClient, error) {
	auth, err := ssh.NewPublicKeysFromFile("git", sshKey, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH auth: %w", err)
	}

	repo, err := git.PlainClone("./repo", false, &git.CloneOptions{
		URL:  sshURL,
		Auth: auth,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	return &GitClient{repo: repo, auth: auth}, nil
}

func (gc *GitClient) CheckoutNewBranch(baseBranch, branchPrefix string) (string, error) {
	w, err := gc.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(baseBranch),
	})
	if err != nil {
		return "", fmt.Errorf("failed to checkout base branch: %w", err)
	}

	newBranch := fmt.Sprintf("%s%s", branchPrefix, time.Now().Format("20060102150405"))
	err = w.Checkout(&git.CheckoutOptions{
		Create: true,
		Branch: plumbing.NewBranchReferenceName(newBranch),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create new branch: %w", err)
	}

	return newBranch, nil
}

func (gc *GitClient) CommitAndPush(branchName, sshUsername, sshEmail string) error {
	w, err := gc.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	_, err = w.Commit("Update Grafana dashboards", &git.CommitOptions{All: true, Author: &object.Signature{
		Name:  sshUsername,
		Email: sshEmail,
		When:  time.Now(),
	}})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	err = gc.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", branchName, branchName))},
		Auth:       gc.auth,
	})
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}
