package git

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"testing"
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
		signer, err = parseSSHPrivateKey(sshKey)
	} else {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(sshKey, []byte(sshKeyPassword))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key: %w", err)
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

func generateSSHKey(t *testing.T, keyType string) ([]byte, error) {
	t.Helper()

	var privateKey interface{}
	var err error

	switch keyType {
	case "rsa":
		privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	case "ecdsa":
		privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "ed25519":
		_, privateKey, err = ed25519.GenerateKey(rand.Reader)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate %s key: %w", keyType, err)
	}

	var pemData []byte

	switch k := privateKey.(type) {
	case *rsa.PrivateKey:
		pemData = pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(k),
		})
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ECDSA key: %w", err)
		}
		pemData = pem.EncodeToMemory(&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: b,
		})
	case ed25519.PrivateKey:
		pemData, err = marshalOpenSSHED25519PrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Ed25519 key: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported key type: %T", k)
	}

	return pemData, nil
}

func marshalOpenSSHED25519PrivateKey(privateKey ed25519.PrivateKey) ([]byte, error) {
	pubKey := privateKey.Public().(ed25519.PublicKey)

	keyBytes := []byte("openssh-key-v1\x00")

	// Cipher, KDF, KDF options (all empty for unencrypted key)
	keyBytes = append(keyBytes, 0, 0, 0, 4) // 4 bytes for "none"
	keyBytes = append(keyBytes, []byte("none")...)
	keyBytes = append(keyBytes, 0, 0, 0, 4) // 4 bytes for "none"
	keyBytes = append(keyBytes, []byte("none")...)
	keyBytes = append(keyBytes, 0, 0, 0, 0) // 4 bytes for empty KDF options

	keyBytes = append(keyBytes, 0, 0, 0, 1) // 4 bytes for number of keys (1)

	// Public key
	pubKeyBytes := ssh.Marshal(struct {
		KeyType string
		PubKey  []byte
	}{
		KeyType: ssh.KeyAlgoED25519,
		PubKey:  pubKey,
	})

	keyBytes = binary.BigEndian.AppendUint32(keyBytes, uint32(len(pubKeyBytes)))
	keyBytes = append(keyBytes, pubKeyBytes...)

	// Generate random check integers
	checkInt := make([]byte, 4)
	_, err := rand.Read(checkInt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random check integers: %w", err)
	}

	// Private key
	privKeyBytes := ssh.Marshal(struct {
		CheckInt1  uint32
		CheckInt2  uint32
		KeyType    string
		PubKey     []byte
		PrivKeyPad []byte
		Comment    string
	}{
		CheckInt1:  binary.BigEndian.Uint32(checkInt),
		CheckInt2:  binary.BigEndian.Uint32(checkInt),
		KeyType:    ssh.KeyAlgoED25519,
		PubKey:     pubKey,
		PrivKeyPad: privateKey,
		Comment:    "",
	})

	padding := 8 - (len(privKeyBytes) % 8)
	for i := 0; i < padding; i++ {
		privKeyBytes = append(privKeyBytes, byte(i+1))
	}

	keyBytes = binary.BigEndian.AppendUint32(keyBytes, uint32(len(privKeyBytes)))
	keyBytes = append(keyBytes, privKeyBytes...)

	pemBlock := &pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: keyBytes,
	}

	return pem.EncodeToMemory(pemBlock), nil
}

func parseSSHPrivateKey(privateKey []byte) (ssh.Signer, error) {
	return ssh.ParsePrivateKey(privateKey)
}

func marshalOpenSSHPrivateKey(signer ssh.Signer) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: signer.PublicKey().Marshal(),
	})
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
