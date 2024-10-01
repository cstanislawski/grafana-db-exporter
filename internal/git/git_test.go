package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestNew(t *testing.T) {
	tempDir := t.TempDir()

	keyTypes := []string{"rsa", "ecdsa", "ed25519"}

	for _, keyType := range keyTypes {
		t.Run(fmt.Sprintf("SSH_%s", strings.ToUpper(keyType)), func(t *testing.T) {
			sshKeyPath := filepath.Join(tempDir, fmt.Sprintf("id_%s", keyType))
			sshKeyContent, err := generateSSHKey(t, keyType)
			if err != nil {
				t.Fatalf("Failed to generate %s SSH key: %v", keyType, err)
			}

			err = os.WriteFile(sshKeyPath, sshKeyContent, 0600)
			if err != nil {
				t.Fatalf("Failed to write %s SSH key: %v", keyType, err)
			}

			knownHostsPath := filepath.Join(tempDir, "known_hosts")
			err = os.WriteFile(knownHostsPath, []byte("github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"), 0600)
			if err != nil {
				t.Fatalf("Failed to write dummy known hosts: %v", err)
			}

			mockRepo := filepath.Join(tempDir, fmt.Sprintf("mock_repo_%s", keyType))
			err = os.MkdirAll(mockRepo, 0755)
			if err != nil {
				t.Fatalf("Failed to create mock repository: %v", err)
			}

			repo, err := git.PlainInit(mockRepo, false)
			if err != nil {
				t.Fatalf("Failed to initialize mock repository: %v", err)
			}

			w, err := repo.Worktree()
			if err != nil {
				t.Fatalf("Failed to get worktree: %v", err)
			}

			dummyFile := filepath.Join(mockRepo, "dummy.txt")
			err = os.WriteFile(dummyFile, []byte("dummy content"), 0644)
			if err != nil {
				t.Fatalf("Failed to create dummy file: %v", err)
			}

			_, err = w.Add("dummy.txt")
			if err != nil {
				t.Fatalf("Failed to add dummy file: %v", err)
			}

			_, err = w.Commit("Initial commit", &git.CommitOptions{
				Author: &object.Signature{
					Name:  "Test User",
					Email: "test@example.com",
					When:  time.Now(),
				},
			})
			if err != nil {
				t.Fatalf("Failed to create initial commit: %v", err)
			}

			client, err := New(filepath.Join(tempDir, fmt.Sprintf("repo_%s", keyType)), mockRepo, sshKeyPath, "", knownHostsPath, true)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			if client == nil {
				t.Errorf("New() returned nil client")
			}

			if client.auth == nil {
				t.Errorf("New() did not create auth method")
			}

			clonedDummyFile := filepath.Join(filepath.Join(tempDir, fmt.Sprintf("repo_%s", keyType)), "dummy.txt")
			if _, err := os.Stat(clonedDummyFile); os.IsNotExist(err) {
				t.Errorf("Cloned repository does not contain the expected dummy file")
			}
		})
	}
}

func TestClient_CheckoutNewBranch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "git-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = worktree.Add("test.txt")
	if err != nil {
		t.Fatalf("Failed to add test file: %v", err)
	}

	_, err = worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	headRef, err := repo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD reference: %v", err)
	}

	err = repo.CreateBranch(&config.Branch{
		Name:   "main",
		Remote: "origin",
		Merge:  headRef.Name(),
	})
	if err != nil {
		t.Fatalf("Failed to create main branch: %v", err)
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("main"),
		Create: true,
	})
	if err != nil {
		t.Fatalf("Failed to checkout main branch: %v", err)
	}

	client := &Client{repo: repo}

	tests := []struct {
		name         string
		baseBranch   string
		branchPrefix string
		wantErr      bool
	}{
		{
			name:         "Valid new branch",
			baseBranch:   "main",
			branchPrefix: "test-",
			wantErr:      false,
		},
		{
			name:         "Invalid base branch",
			baseBranch:   "nonexistent",
			branchPrefix: "test-",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branchName, err := client.CheckoutNewBranch(context.Background(), tt.baseBranch, tt.branchPrefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckoutNewBranch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !branchExists(t, repo, branchName) {
					t.Errorf("Branch %s was not created", branchName)
				}
			}
		})
	}
}

func TestClient_CommitAll(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "git-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	client := &Client{repo: repo}

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = client.CommitAll(context.Background(), "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("CommitAll() error = %v", err)
	}

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("Failed to get commit object: %v", err)
	}

	if commit.Author.Name != "testuser" || commit.Author.Email != "test@example.com" {
		t.Errorf("Commit author mismatch. Got %s <%s>, want testuser <test@example.com>", commit.Author.Name, commit.Author.Email)
	}

	if commit.Message != "Update Grafana dashboards" {
		t.Errorf("Commit message mismatch. Got %s, want 'Update Grafana dashboards'", commit.Message)
	}
}

func TestClient_Push(t *testing.T) {
	// for now, we're just testing that the method doesn't return an error when there's no remote, as it's challenging to mock the push
	tempDir, err := os.MkdirTemp("", "git-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	client := &Client{repo: repo}

	err = client.Push(context.Background(), "main")
	if err == nil {
		t.Errorf("Push() should return an error when there's no remote")
	}
}

func branchExists(t *testing.T, repo *git.Repository, branchName string) bool {
	t.Helper()
	branches, err := repo.Branches()
	if err != nil {
		t.Fatalf("Failed to get branches: %v", err)
	}
	exists := false
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().Short() == branchName {
			exists = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to iterate branches: %v", err)
	}
	return exists
}
