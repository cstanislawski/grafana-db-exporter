package git

import (
	"context"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	gogitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Client struct {
	repo *git.Repository
	auth *gogitssh.PublicKeys
}

func New(repoClonePath, sshURL, sshKeyPath, sshKeyPassword, knownHostsPath string, allowUnknownHosts bool) (*Client, error) {
	sshKey, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key: %w", err)
	}

	sshKey = []byte(strings.TrimSpace(string(sshKey)))

	var signer ssh.Signer
	if sshKeyPassword == "" {
		signer, err = ssh.ParsePrivateKey(sshKey)
		if err != nil {
			key, err := x509.ParsePKCS8PrivateKey(sshKey)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SSH key: %w", err)
			}
			signer, err = ssh.NewSignerFromKey(key)
			if err != nil {
				return nil, fmt.Errorf("failed to create signer from parsed key: %w", err)
			}
		}
	} else {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(sshKey, []byte(sshKeyPassword))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key: %w, key content: %s", err, string(sshKey))
	}

	auth := &gogitssh.PublicKeys{User: "git", Signer: signer}

	if allowUnknownHosts {
		auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("known hosts file does not exist and allowUnknownHosts is false: %w", err)
		}
		hostKeyCallback, err := knownhosts.New(knownHostsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create known hosts callback: %w", err)
		}
		auth.HostKeyCallback = hostKeyCallback
	}

	repo, err := git.PlainClone(repoClonePath, false, &git.CloneOptions{
		URL:  sshURL,
		Auth: auth,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	return &Client{repo: repo, auth: auth}, nil
}

func (gc *Client) CheckoutNewBranch(ctx context.Context, baseBranch, branchPrefix string) (string, error) {
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

func (gc *Client) CommitAll(ctx context.Context, sshUsername, sshEmail string) error {
	w, err := gc.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	_, err = w.Add(".")
	if err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	_, err = w.Commit("Update Grafana dashboards", &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name:  sshUsername,
			Email: sshEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

func (gc *Client) Push(ctx context.Context, branchName string) error {
	err := gc.repo.PushContext(ctx, &git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", branchName, branchName))},
		Auth:       gc.auth,
	})
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}
