package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/farcloser/lepton/pkg/testutil"
)

type CosignKeyPair struct {
	PublicKey  string
	PrivateKey string
	Cleanup    func()
}

func NewCosignKeyPair(t testing.TB, path string, password string) *CosignKeyPair {
	td, err := os.MkdirTemp(t.TempDir(), path)
	assert.NilError(t, err)

	cmd := exec.Command("cosign", "generate-key-pair")
	cmd.Dir = td
	cmd.Env = append(cmd.Env, fmt.Sprintf("COSIGN_PASSWORD=%s", password))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to run %v: %v (%q)", cmd.Args, err, string(out))
	}

	publicKey := filepath.Join(td, "cosign.pub")
	privateKey := filepath.Join(td, "cosign.key")

	return &CosignKeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Cleanup: func() {
			_ = os.RemoveAll(td)
		},
	}
}

func CreateBuildContext(t *testing.T, dockerfile string) string {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte(dockerfile), 0644)
	assert.NilError(t, err)
	return tmpDir
}

func RequiresSoci(base *testutil.Base) {
	info := base.Info()
	for _, p := range info.Plugins.Storage {
		if p == "soci" {
			return
		}
	}
	base.T.Skip("test requires soci")
}

type DockerArchiveManifestJSON []DockerArchiveManifestJSONEntry

type DockerArchiveManifestJSONEntry struct {
	Config   string
	RepoTags []string
	Layers   []string
}

func ExtractDockerArchive(archiveTarPath, rootfsPath string) error {
	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		return err
	}
	workDir, err := os.MkdirTemp("", "extract-docker-archive")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)
	if err := ExtractTarFile(workDir, archiveTarPath); err != nil {
		return err
	}
	manifestJSONPath := filepath.Join(workDir, "manifest.json")
	manifestJSONBytes, err := os.ReadFile(manifestJSONPath)
	if err != nil {
		return err
	}
	var mani DockerArchiveManifestJSON
	if err := json.Unmarshal(manifestJSONBytes, &mani); err != nil {
		return err
	}
	if len(mani) > 1 {
		return fmt.Errorf("multi-image archive cannot be extracted: contains %d images", len(mani))
	}
	if len(mani) < 1 {
		return errors.New("invalid archive")
	}
	ent := mani[0]
	for _, l := range ent.Layers {
		layerTarPath := filepath.Join(workDir, l)
		if err := ExtractTarFile(rootfsPath, layerTarPath); err != nil {
			return err
		}
	}
	return nil
}

func FindIPv6(output string) net.IP {
	var ipv6 string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "inet6") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				ipv6 = strings.Split(fields[1], "/")[0]
				break
			}
		}
	}
	return net.ParseIP(ipv6)
}
