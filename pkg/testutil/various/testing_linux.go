/*
   Copyright Farcloser.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package various

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nettestutil"
)

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

type JweKeyPair struct {
	Prv     string
	Pub     string
	Cleanup func()
}

func NewJWEKeyPair(t testing.TB) *JweKeyPair {
	testutil.RequireExecutable(t, "openssl")
	td := t.TempDir()
	prv := filepath.Join(td, "mykey.pem")
	pub := filepath.Join(td, "mypubkey.pem")
	cmds := [][]string{
		// Exec openssl commands to ensure that nerdctl is compatible with the output of openssl commands.
		// Do NOT refactor this function to use "crypto/rsa" stdlib.
		{"openssl", "genrsa", "-out", prv},
		{"openssl", "rsa", "-in", prv, "-pubout", "-out", pub},
	}
	for _, f := range cmds {
		cmd := exec.Command(f[0], f[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to run %v: %v (%q)", cmd.Args, err, string(out))
		}
	}
	return &JweKeyPair{
		Prv: prv,
		Pub: pub,
		Cleanup: func() {
			_ = os.RemoveAll(td)
		},
	}
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

type CosignKeyPair struct {
	PublicKey  string
	PrivateKey string
	Cleanup    func()
}

func NewCosignKeyPair(t testing.TB, path, password string) *CosignKeyPair {
	td := t.TempDir()

	cmd := exec.Command("cosign", "generate-key-pair")
	cmd.Dir = td
	cmd.Env = append(cmd.Env, "COSIGN_PASSWORD="+password)
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

func ComposeUp(
	t *testing.T,
	base *testutil.Base,
	port string,
	dockerComposeYAML string,
	opts ...string,
) {
	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	t.Cleanup(func() {
		comp.CleanUp()
	})

	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)

	base.ComposeCmd(append(append([]string{"-f", comp.YAMLFullPath()}, opts...), "up", "-d")...).
		AssertOK()
	t.Cleanup(func() {
		base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()
	})
	base.Cmd("volume", "inspect", projectName+"_db").AssertOK()
	base.Cmd("network", "inspect", projectName+"_default").AssertOK()

	checkWordpress := func() error {
		resp, err := nettestutil.HTTPGet("http://127.0.0.1:"+port, 10, false)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if !strings.Contains(string(respBody), testutil.WordpressIndexHTMLSnippet) {
			t.Logf("respBody=%q", respBody)
			return fmt.Errorf("respBody does not contain %q", testutil.WordpressIndexHTMLSnippet)
		}
		return nil
	}

	var wordpressWorking bool
	for i := range 30 {
		t.Logf("(retry %d)", i)
		err := checkWordpress()
		if err == nil {
			wordpressWorking = true
			break
		}
		// NOTE: "<h1>Error establishing a database connection</h1>" is expected for the first few iterations
		t.Log(err)
		time.Sleep(3 * time.Second)
	}

	if !wordpressWorking {
		t.Fatal("wordpress is not working")
	}
	t.Log("wordpress seems functional")

	base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").AssertOK()
	base.Cmd("volume", "inspect", projectName+"_db").AssertFail()
	base.Cmd("network", "inspect", projectName+"_default").AssertFail()
}
