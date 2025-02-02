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

package testregistry

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"go.farcloser.world/tigron/test"
	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/containerd/nerdctl/v2/pkg/testutil"
	"github.com/containerd/nerdctl/v2/pkg/testutil/nerdtest/platform"
	"github.com/containerd/nerdctl/v2/pkg/testutil/nettestutil"
	"github.com/containerd/nerdctl/v2/pkg/testutil/portlock"
	"github.com/containerd/nerdctl/v2/pkg/testutil/testca"
)

type RegistryServer struct {
	IP       net.IP
	Port     int
	Scheme   string
	ListenIP net.IP
	Cleanup  func(err error)
	Logs     func()
	HostsDir string // contains "<HostIP>:<ListenPort>/hosts.toml"
}

type TokenAuthServer struct {
	IP       net.IP
	Port     int
	Scheme   string
	ListenIP net.IP
	Cleanup  func(err error)
	Logs     func()
	Auth     Auth
	CertPath string
}

func ensureImages(_ test.Data, helpers test.Helpers) {
	registryImage := platform.RegistryImageStable

	up := os.Getenv("DISTRIBUTION_VERSION")
	if up != "" {
		if up[0:1] != "v" {
			up = "v" + up
		}
		registryImage = platform.RegistryImageNext + up
	}

	helpers.Ensure("pull", "--quiet", registryImage)
	helpers.Ensure("pull", "--quiet", platform.DockerAuthImage)
}

func NewAuthServer(data test.Data, helpers test.Helpers, ca *testca.CA, port int, user, pass string, tls bool) *TokenAuthServer {
	ensureImages(data, helpers)
	name := data.Identifier()
	// listen on 0.0.0.0 to enable 127.0.0.1
	listenIP := net.ParseIP("0.0.0.0")
	hostIP, err := nettestutil.NonLoopbackIPv4()
	assert.NilError(helpers.T(), err, fmt.Errorf("failed finding ipv4 non loopback interface: %w", err))
	// Prepare configuration file for authentication server
	// Details: https://github.com/cesanta/docker_auth/blob/1.7.1/examples/simple.yml
	configFile, err := os.CreateTemp("", "authconfig")
	assert.NilError(helpers.T(), err, fmt.Errorf("failed creating temporary directory for config file: %w", err))
	bpass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	assert.NilError(helpers.T(), err, fmt.Errorf("failed bcrypt encrypting password: %w", err))
	configFileName := configFile.Name()
	scheme := "http"
	configContent := fmt.Sprintf(`
server:
  addr: ":5100"
token:
  issuer: "Acme auth server"
  expiration: 900
  certificate: "/auth/domain.crt"
  key: "/auth/domain.key"
users:
  "%s":
    password: "%s"
acl:
  - match: {account: "%s"}
    actions: ["*"]
`, user, string(bpass), user)
	if tls {
		scheme = "https"
		configContent = fmt.Sprintf(`
server:
  addr: ":5100"
  certificate: "/auth/domain.crt"
  key: "/auth/domain.key"
token:
  issuer: "Acme auth server"
  expiration: 900
users:
  "%s":
    password: "%s"
acl:
  - match: {account: "%s"}
    actions: ["*"]
`, user, string(bpass), user)
	}
	_, err = configFile.WriteString(configContent)
	assert.NilError(helpers.T(), err, fmt.Errorf("failed writing configuration: %w", err))

	cert := ca.NewCert(hostIP.String())

	port, err = portlock.Acquire(port)
	assert.NilError(helpers.T(), err, fmt.Errorf("failed acquiring port: %w", err))
	containerName := fmt.Sprintf("auth-%s-%d", name, port)
	// Cleanup possible leftovers first
	helpers.Anyhow("rm", "-f", containerName)

	cleanup := func(err error) {
		helpers.Anyhow("rm", "-f", containerName)
		errPortRelease := portlock.Release(port)
		errCertClose := cert.Close()
		errConfigClose := configFile.Close()
		errConfigRemove := os.Remove(configFileName)
		if err == nil {
			// assert.NilError(helpers.T(), result.Error, fmt.Errorf("failed stopping container: %w", err))
			assert.NilError(helpers.T(), errPortRelease, fmt.Errorf("failed releasing port: %w", err))
			assert.NilError(helpers.T(), errCertClose, fmt.Errorf("failed cleaning certs: %w", err))
			assert.NilError(helpers.T(), errConfigClose, fmt.Errorf("failed closing config file: %w", err))
			assert.NilError(helpers.T(), errConfigRemove, fmt.Errorf("failed removing config file: %w", err))
		}
	}

	err = func() error {
		// Run authentication server
		helpers.Ensure(
			"run",
			"--pull=never",
			"-d",
			"-p", fmt.Sprintf("%s:%d:5100", listenIP, port),
			"--name", containerName,
			"-v", cert.CertPath+":/auth/domain.crt",
			"-v", cert.KeyPath+":/auth/domain.key",
			"-v", configFileName+":/config/auth_config.yml",
			testutil.DockerAuthImage,
			"/config/auth_config.yml")

		joined := net.JoinHostPort(hostIP.String(), strconv.Itoa(port))
		_, err = nettestutil.HTTPGet(fmt.Sprintf("%s://%s/auth", scheme, joined), 30, true)
		return err
	}()
	if err != nil {
		cl := helpers.Capture("logs", containerName)
		helpers.T().Logf("%s:\n%s\n", containerName, cl)
		cleanup(err)
	}
	assert.NilError(helpers.T(), err, fmt.Errorf("failed starting auth container in a timely manner: %w", err))

	return &TokenAuthServer{
		IP:       hostIP,
		Port:     port,
		Scheme:   scheme,
		ListenIP: listenIP,
		CertPath: cert.CertPath,
		Auth: &TokenAuth{
			Address:  scheme + "://" + net.JoinHostPort(hostIP.String(), strconv.Itoa(port)),
			CertPath: cert.CertPath,
		},
		Cleanup: cleanup,
		Logs: func() {
			helpers.T().Logf("%s: %q", containerName, helpers.Capture("logs", containerName))
		},
	}
}

// Auth is an interface to pass to the test registry for configuring authentication
type Auth interface {
	Params(data test.Data, helpers test.Helpers) []string
}

type NoAuth struct {
}

func (na *NoAuth) Params(data test.Data, helpers test.Helpers) []string {
	return []string{}
}

type TokenAuth struct {
	Address  string
	CertPath string
}

func (ta *TokenAuth) Params(data test.Data, helpers test.Helpers) []string {
	return []string{
		"--env", "REGISTRY_AUTH=token",
		"--env", "REGISTRY_AUTH_TOKEN_REALM=" + ta.Address + "/auth",
		"--env", "REGISTRY_AUTH_TOKEN_SERVICE=Docker registry",
		"--env", "REGISTRY_AUTH_TOKEN_ISSUER=Acme auth server",
		"--env", "REGISTRY_AUTH_TOKEN_ROOTCERTBUNDLE=/auth/domain.crt",
		"-v", ta.CertPath + ":/auth/domain.crt",
	}
}

type BasicAuth struct {
	Realm    string
	HtFile   string
	Username string
	Password string
}

func (ba *BasicAuth) Params(data test.Data, helpers test.Helpers) []string {
	if ba.Realm == "" {
		ba.Realm = "Basic Realm"
	}
	if ba.HtFile == "" && ba.Username != "" && ba.Password != "" {
		pass := ba.Password
		encryptedPass, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		tmpDir, _ := os.MkdirTemp(helpers.T().TempDir(), "htpasswd")
		ba.HtFile = filepath.Join(tmpDir, "htpasswd")
		_ = os.WriteFile(ba.HtFile, []byte(fmt.Sprintf(`%s:%s`, ba.Username, string(encryptedPass))), 0o600)
	}
	ret := []string{
		"--env", "REGISTRY_AUTH=htpasswd",
		"--env", "REGISTRY_AUTH_HTPASSWD_REALM=" + ba.Realm,
		"--env", "REGISTRY_AUTH_HTPASSWD_PATH=/htpasswd",
	}
	if ba.HtFile != "" {
		ret = append(ret, "-v", ba.HtFile+":/htpasswd")
	}
	return ret
}

func NewRegistry(data test.Data, helpers test.Helpers, ca *testca.CA, port int, auth Auth, boundCleanup func(error)) *RegistryServer {
	ensureImages(data, helpers)

	name := data.Identifier()
	// listen on 0.0.0.0 to enable 127.0.0.1
	listenIP := net.ParseIP("0.0.0.0")
	hostIP, err := nettestutil.NonLoopbackIPv4()
	assert.NilError(helpers.T(), err, fmt.Errorf("failed finding ipv4 non loopback interface: %w", err))
	port, err = portlock.Acquire(port)
	assert.NilError(helpers.T(), err, fmt.Errorf("failed acquiring port: %w", err))

	containerName := fmt.Sprintf("registry-%s-%d", name, port)
	// Cleanup possible leftovers first
	helpers.Anyhow("rm", "-f", containerName)

	args := []string{
		"run",
		"--pull=never",
		"-d",
		"-p", fmt.Sprintf("%s:%d:5000", listenIP, port),
		"--name", containerName,
	}
	scheme := "http"
	var cert *testca.Cert
	if ca != nil {
		scheme = "https"
		cert = ca.NewCert(hostIP.String(), "127.0.0.1", "localhost", "::1")
		args = append(args,
			"--env", "REGISTRY_HTTP_TLS_CERTIFICATE=/registry/domain.crt",
			"--env", "REGISTRY_HTTP_TLS_KEY=/registry/domain.key",
			"-v", cert.CertPath+":/registry/domain.crt",
			"-v", cert.KeyPath+":/registry/domain.key",
		)
	}

	args = append(args, auth.Params(data, helpers)...)
	registryImage := testutil.RegistryImageStable

	up := os.Getenv("DISTRIBUTION_VERSION")
	if up != "" {
		if up[0:1] != "v" {
			up = "v" + up
		}
		registryImage = testutil.RegistryImageNext + up
	}
	args = append(args, registryImage)

	cleanup := func(err error) {
		helpers.Anyhow("rm", "-f", containerName)
		errPortRelease := portlock.Release(port)
		var errCertClose error
		if cert != nil {
			errCertClose = cert.Close()
		}
		if boundCleanup != nil {
			boundCleanup(err)
		}
		if cert != nil && err == nil {
			assert.NilError(helpers.T(), errCertClose, fmt.Errorf("failed cleaning certificates: %w", err))
		}
		if err == nil {
			assert.NilError(helpers.T(), errPortRelease, fmt.Errorf("failed releasing port: %w", err))
		}
	}

	hostsDir, err := func() (string, error) {
		hDir, err := os.MkdirTemp(helpers.T().TempDir(), "certs.d")
		if err != nil {
			return "", err
		}

		if ca != nil {
			err = generateCertsd(hDir, ca.CertPath, hostIP.String(), port)
			if err != nil {
				return "", err
			}
			err = generateCertsd(hDir, ca.CertPath, "127.0.0.1", port)
			if err != nil {
				return "", err
			}
			err = generateCertsd(hDir, ca.CertPath, "localhost", port)
			if err != nil {
				return "", err
			}
			if port == 443 {
				err = generateCertsd(hDir, ca.CertPath, hostIP.String(), 0)
				if err != nil {
					return "", err
				}
				err = generateCertsd(hDir, ca.CertPath, "127.0.0.1", 0)
				if err != nil {
					return "", err
				}
				err = generateCertsd(hDir, ca.CertPath, "localhost", 0)
				if err != nil {
					return "", err
				}
			}
		}

		helpers.Ensure(args...)

		if _, err = nettestutil.HTTPGet(fmt.Sprintf("%s://%s:%s/v2", scheme, hostIP.String(), strconv.Itoa(port)), 30, true); err != nil {
			return "", err
		}

		return hDir, nil
	}()
	if err != nil {
		cl := helpers.Capture("logs", containerName)
		helpers.T().Logf("%s:\n%s\n", containerName, cl)
		cleanup(err)
	}
	assert.NilError(helpers.T(), err, fmt.Errorf("failed starting registry container in a timely manner: %w", err))

	output := helpers.Capture("logs", containerName)
	return &RegistryServer{
		IP:       hostIP,
		Port:     port,
		Scheme:   scheme,
		ListenIP: listenIP,
		Cleanup:  cleanup,
		Logs: func() {
			helpers.T().Logf("%s: %q", containerName, output)
		},
		HostsDir: hostsDir,
	}
}

func NewWithTokenAuth(data test.Data, helpers test.Helpers, user, pass string, port int, tls bool) *RegistryServer {
	ca := testca.New(data, helpers)
	as := NewAuthServer(data, helpers, ca, 0, user, pass, tls)
	auth := &TokenAuth{
		Address:  as.Scheme + "://" + net.JoinHostPort(as.IP.String(), strconv.Itoa(as.Port)),
		CertPath: as.CertPath,
	}
	return NewRegistry(data, helpers, ca, port, auth, as.Cleanup)
}

func NewWithNoAuth(data test.Data, helpers test.Helpers, port int, tls bool) *RegistryServer {
	var ca *testca.CA
	if tls {
		ca = testca.New(data, helpers)
	}
	return NewRegistry(data, helpers, ca, port, &NoAuth{}, nil)
}
