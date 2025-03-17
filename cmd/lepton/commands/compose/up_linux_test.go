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

package compose_test

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/containerd/log"
	"github.com/docker/go-connections/nat"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"

	"go.farcloser.world/lepton/pkg/composer/serviceparser"
	"go.farcloser.world/lepton/pkg/rootlessutil"
	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nettestutil"
	"go.farcloser.world/lepton/pkg/testutil/various"
)

func TestComposeUp(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	various.ComposeUp(t, base, "8087", fmt.Sprintf(`
version: '3.1'

services:

  wordpress:
    image: %s
    restart: always
    ports:
      - 8087:80
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: exampleuser
      WORDPRESS_DB_PASSWORD: examplepass
      WORDPRESS_DB_NAME: exampledb
    volumes:
      - wordpress:/var/www/html

  db:
    image: %s
    restart: always
    environment:
      MYSQL_DATABASE: exampledb
      MYSQL_USER: exampleuser
      MYSQL_PASSWORD: examplepass
      MYSQL_RANDOM_ROOT_PASSWORD: '1'
    volumes:
      - db:/var/lib/mysql

volumes:
  wordpress:
  db:
`, testutil.WordpressImage, testutil.MariaDBImage))
}

func TestComposeUpBuild(t *testing.T) {
	t.Parallel()

	testutil.RequiresBuild(t)
	testutil.RegisterBuildCacheCleanup(t)
	base := testutil.NewBase(t)

	const dockerComposeYAML = `
services:
  web:
    build: .
    ports:
    - 8083:80
`
	dockerfile := fmt.Sprintf(`FROM %s
COPY index.html /usr/share/nginx/html/index.html
`, testutil.NginxAlpineImage)
	indexHTML := t.Name()

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)

	comp.WriteFile("Dockerfile", dockerfile)
	comp.WriteFile("index.html", indexHTML)

	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d", "--build").AssertOK()
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	resp, err := nettestutil.HTTPGet("http://127.0.0.1:8083", 50, false)
	assert.NilError(t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	assert.NilError(t, err)
	t.Logf("respBody=%q", respBody)
	assert.Assert(t, strings.Contains(string(respBody), indexHTML))
}

func TestComposeUpNetWithStaticIP(t *testing.T) {
	t.Parallel()

	if rootlessutil.IsRootless() {
		t.Skip("Static IP assignment is not supported rootless mode yet.")
	}
	base := testutil.NewBase(t)
	staticIP := "172.20.0.12"
	var dockerComposeYAML = fmt.Sprintf(`
version: '3.1'

services:
  svc0:
    image: %s
    networks:
      net0:
        ipv4_address: %s

networks:
  net0:
    ipam:
      config:
        - subnet: 172.20.0.0/24
`, testutil.NginxAlpineImage, staticIP)
	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)
	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d").AssertOK()
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	svc0 := serviceparser.DefaultContainerName(projectName, "svc0", "1")
	inspectCmd := base.Cmd("inspect", svc0, "--format", "\"{{range .NetworkSettings.Networks}} {{.IPAddress}}{{end}}\"")
	result := inspectCmd.Run()
	stdoutContent := result.Stdout() + result.Stderr()
	assert.Assert(inspectCmd.Base.T, result.ExitCode == 0, stdoutContent)
	if !strings.Contains(stdoutContent, staticIP) {
		log.L.Errorf("test failed, the actual container ip is %s", stdoutContent)
		t.Fail()
		return
	}
}

func TestComposeUpMultiNet(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)

	var dockerComposeYAML = fmt.Sprintf(`
version: '3.1'

services:
  svc0:
    image: %s
    networks:
      - net0
      - net1
      - net2
  svc1:
    image: %s
    networks:
      - net0
      - net1
  svc2:
    image: %s
    networks:
      - net2

networks:
  net0: {}
  net1: {}
  net2: {}
`, testutil.NginxAlpineImage, testutil.NginxAlpineImage, testutil.NginxAlpineImage)
	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()

	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)

	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d").AssertOK()
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	svc0 := serviceparser.DefaultContainerName(projectName, "svc0", "1")
	svc1 := serviceparser.DefaultContainerName(projectName, "svc1", "1")
	svc2 := serviceparser.DefaultContainerName(projectName, "svc2", "1")

	base.Cmd("exec", svc0, "ping", "-c", "1", "svc0").AssertOK()
	base.Cmd("exec", svc0, "ping", "-c", "1", "svc1").AssertOK()
	base.Cmd("exec", svc0, "ping", "-c", "1", "svc2").AssertOK()
	base.Cmd("exec", svc1, "ping", "-c", "1", "svc0").AssertOK()
	base.Cmd("exec", svc2, "ping", "-c", "1", "svc0").AssertOK()
	base.Cmd("exec", svc1, "ping", "-c", "1", "svc2").AssertFail()
}

func TestComposeUpOsEnvVar(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	const containerName = "nginxAlpine"
	var dockerComposeYAML = fmt.Sprintf(`
version: '3.1'

services:
  svc1:
    image: %s
    container_name: %s
    ports:
      - ${ADDRESS:-127.0.0.1}:8084:80
`, testutil.NginxAlpineImage, containerName)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)

	base.Env = append(base.Env, "ADDRESS=0.0.0.0")

	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d").AssertOK()
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	inspect := base.InspectContainer(containerName)
	inspect80TCP := (*inspect.NetworkSettings.Ports)["80/tcp"]
	expected := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: "8084",
	}
	assert.Equal(base.T, expected, inspect80TCP[0])
}

func TestComposeUpDotEnvFile(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)

	var dockerComposeYAML = `
version: '3.1'

services:
  svc3:
    image: ghcr.io/stargz-containers/nginx:$TAG
`

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)

	envFile := `TAG=1.19-alpine-org`
	comp.WriteFile(".env", envFile)

	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d").AssertOK()
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()
}

func TestComposeUpEnvFileNotFoundError(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)

	var dockerComposeYAML = `
version: '3.1'

services:
  svc4:
    image: ghcr.io/stargz-containers/nginx:$TAG
`

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)

	envFile := `TAG=1.19-alpine-org`
	comp.WriteFile("envFile", envFile)

	// env-file is relative to the current working directory and not the project directory
	base.ComposeCmd("-f", comp.YAMLFullPath(), "--env-file", "envFile", "up", "-d").AssertFail()
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()
}

func TestComposeUpWithScale(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)

	var dockerComposeYAML = fmt.Sprintf(`
version: '3.1'

services:
  test:
    image: %s
    command: "sleep infinity"
`, testutil.AlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)

	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d", "--scale", "test=2").AssertOK()
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	base.ComposeCmd("-f", comp.YAMLFullPath(), "ps").AssertOutContains(serviceparser.DefaultContainerName(projectName, "test", "2"))
}

func TestComposeIPAMConfig(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)

	var dockerComposeYAML = fmt.Sprintf(`
version: '3.1'

services:
  foo:
    image: %s
    command: "sleep infinity"

networks:
  default:
    ipam:
      config:
        - subnet: 10.43.100.0/24
`, testutil.AlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)

	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d").AssertOK()
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	base.Cmd("inspect", "-f", `{{json .NetworkSettings.Networks }}`,
		serviceparser.DefaultContainerName(projectName, "foo", "1")).AssertOutContains("10.43.100.")
}

func TestComposeUpRemoveOrphans(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)

	var (
		dockerComposeYAMLOrphan = fmt.Sprintf(`
version: '3.1'

services:
  test:
    image: %s
    command: "sleep infinity"
`, testutil.AlpineImage)

		dockerComposeYAMLFull = fmt.Sprintf(`
%s
  orphan:
    image: %s
    command: "sleep infinity"
`, dockerComposeYAMLOrphan, testutil.AlpineImage)
	)

	compOrphan := testutil.NewComposeDir(t, dockerComposeYAMLOrphan)
	defer compOrphan.CleanUp()
	compFull := testutil.NewComposeDir(t, dockerComposeYAMLFull)
	defer compFull.CleanUp()

	projectName := fmt.Sprintf("compose-test-%d", time.Now().Unix())
	t.Logf("projectName=%q", projectName)

	orphanContainer := serviceparser.DefaultContainerName(projectName, "orphan", "1")

	base.ComposeCmd("-p", projectName, "-f", compFull.YAMLFullPath(), "up", "-d").AssertOK()
	defer base.ComposeCmd("-p", projectName, "-f", compFull.YAMLFullPath(), "down", "-v").Run()
	base.ComposeCmd("-p", projectName, "-f", compOrphan.YAMLFullPath(), "up", "-d").AssertOK()
	base.ComposeCmd("-p", projectName, "-f", compFull.YAMLFullPath(), "ps").AssertOutContains(orphanContainer)
	base.ComposeCmd("-p", projectName, "-f", compOrphan.YAMLFullPath(), "up", "-d", "--remove-orphans").AssertOK()
	base.ComposeCmd("-p", projectName, "-f", compFull.YAMLFullPath(), "ps").AssertOutNotContains(orphanContainer)
}

func TestComposeUpIdempotent(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)

	var dockerComposeYAML = fmt.Sprintf(`
version: '3.1'

services:
  test:
    image: %s
    command: "sleep infinity"
`, testutil.AlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)

	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d").AssertOK()
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()
	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d").AssertOK()
	base.ComposeCmd("-f", comp.YAMLFullPath(), "down").AssertOK()
}

func TestComposeUpWithExternalNetwork(t *testing.T) {
	t.Parallel()

	containerName1 := testutil.Identifier(t) + "-1"
	containerName2 := testutil.Identifier(t) + "-2"
	networkName := testutil.Identifier(t) + "-network"
	var dockerComposeYaml1 = fmt.Sprintf(`
version: "3"
services:
  %s:
    image: %s
    container_name: %s
    networks:
      %s:
        aliases:
          - nginx-1
networks:
  %s:
    external: true
`, containerName1, testutil.NginxAlpineImage, containerName1, networkName, networkName)
	var dockerComposeYaml2 = fmt.Sprintf(`
version: "3"
services:
  %s:
    image: %s
    container_name: %s
    networks:
      %s:
        aliases:
          - nginx-2
networks:
  %s:
    external: true
`, containerName2, testutil.NginxAlpineImage, containerName2, networkName, networkName)
	comp1 := testutil.NewComposeDir(t, dockerComposeYaml1)
	defer comp1.CleanUp()
	comp2 := testutil.NewComposeDir(t, dockerComposeYaml2)
	defer comp2.CleanUp()
	base := testutil.NewBase(t)
	// Create the test network
	base.Cmd("network", "create", networkName).AssertOK()
	defer base.Cmd("network", "rm", networkName).Run()
	// Run the first compose
	base.ComposeCmd("-f", comp1.YAMLFullPath(), "up", "-d").AssertOK()
	defer base.ComposeCmd("-f", comp1.YAMLFullPath(), "down", "-v").Run()
	// Run the second compose
	base.ComposeCmd("-f", comp2.YAMLFullPath(), "up", "-d").AssertOK()
	defer base.ComposeCmd("-f", comp2.YAMLFullPath(), "down", "-v").Run()
	// Down the second compose
	base.ComposeCmd("-f", comp2.YAMLFullPath(), "down", "-v").AssertOK()
	// Run the second compose again
	base.ComposeCmd("-f", comp2.YAMLFullPath(), "up", "-d").AssertOK()
	base.Cmd("exec", containerName1, "wget", "-qO-", "http://"+containerName2).AssertOutContains(testutil.NginxAlpineIndexHTMLSnippet)
}

func TestComposeUpWithBypass4netns(t *testing.T) {
	t.Parallel()

	// docker does not support bypass4netns mode
	testutil.DockerIncompatible(t)
	if !rootlessutil.IsRootless() {
		t.Skip("test needs rootless")
	}
	testutil.RequireSystemService(t, "bypass4netnsd")
	base := testutil.NewBase(t)
	various.ComposeUp(t, base, "8085", fmt.Sprintf(`
version: '3.1'

services:

  wordpress:
    image: %s
    restart: always
    ports:
      - 8085:80
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: exampleuser
      WORDPRESS_DB_PASSWORD: examplepass
      WORDPRESS_DB_NAME: exampledb
    volumes:
      - wordpress:/var/www/html
    annotations:
      - nerdctl/bypass4netns=1

  db:
    image: %s
    restart: always
    environment:
      MYSQL_DATABASE: exampledb
      MYSQL_USER: exampleuser
      MYSQL_PASSWORD: examplepass
      MYSQL_RANDOM_ROOT_PASSWORD: '1'
    volumes:
      - db:/var/lib/mysql
    annotations:
      - nerdctl/bypass4netns=1

volumes:
  wordpress:
  db:
`, testutil.WordpressImage, testutil.MariaDBImage))
}

func TestComposeUpProfile(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	serviceRegular := testutil.Identifier(t) + "-regular"
	serviceProfiled := testutil.Identifier(t) + "-profiled"

	dockerComposeYAML := fmt.Sprintf(`
services:
  %s:
    image: %[3]s

  %[2]s:
    image: %[3]s
    profiles:
      - test-profile
`, serviceRegular, serviceProfiled, testutil.NginxAlpineImage)

	// * Test with profile
	//   Should run both the services:
	//     - matching active profile
	//     - one without profile
	comp1 := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp1.CleanUp()
	base.ComposeCmd("-f", comp1.YAMLFullPath(), "--profile", "test-profile", "up", "-d").AssertOK()

	psCmd := base.Cmd("ps", "-a", "--format={{.Names}}")
	psCmd.AssertOutContains(serviceRegular)
	psCmd.AssertOutContains(serviceProfiled)
	base.ComposeCmd("-f", comp1.YAMLFullPath(), "--profile", "test-profile", "down", "-v").AssertOK()

	// * Test without profile
	//   Should run:
	//     - service without profile
	comp2 := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp2.CleanUp()
	base.ComposeCmd("-f", comp2.YAMLFullPath(), "up", "-d").AssertOK()
	defer base.ComposeCmd("-f", comp2.YAMLFullPath(), "down", "-v").AssertOK()

	psCmd = base.Cmd("ps", "-a", "--format={{.Names}}")
	psCmd.AssertOutContains(serviceRegular)
	psCmd.AssertOutNotContains(serviceProfiled)
}

func TestComposeUpAbortOnContainerExit(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	serviceRegular := "regular"
	serviceProfiled := "exited"
	dockerComposeYAML := fmt.Sprintf(`
services:
  %s:
    image: %s
    ports:
      - 8086:80
  %s:
    image: %s
    entrypoint: /bin/sh -c "exit 1"
`, serviceRegular, testutil.NginxAlpineImage, serviceProfiled, testutil.BusyboxImage)
	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()

	// here we run 'compose up --abort-on-container-exit' command
	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "--abort-on-container-exit").AssertExitCode(1)
	time.Sleep(3 * time.Second)
	psCmd := base.Cmd("ps", "-a", "--format={{.Names}}", "--filter", "status=exited")

	psCmd.AssertOutContains(serviceRegular)
	psCmd.AssertOutContains(serviceProfiled)
	base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").AssertOK()

	// this time we run 'compose up' command without --abort-on-container-exit flag
	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d").AssertOK()
	time.Sleep(3 * time.Second)
	psCmd = base.Cmd("ps", "-a", "--format={{.Names}}", "--filter", "status=exited")

	// this time the regular service should not be listed in the output
	psCmd.AssertOutNotContains(serviceRegular)
	psCmd.AssertOutContains(serviceProfiled)
	base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").AssertOK()

	// in this subtest we are ensuring that flags '-d' and '--abort-on-container-exit' cannot be run together
	c := base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d", "--abort-on-container-exit")
	expected := icmd.Expected{
		ExitCode: 1,
	}
	c.Assert(expected)
}

func TestComposeUpPull(t *testing.T) {
	// This test is removing the common image
	// t.Parallel()

	base := testutil.NewBase(t)

	var dockerComposeYAML = fmt.Sprintf(`
services:
  test:
    image: %s
    command: sh -euxc "echo hi"
`, testutil.CommonImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()

	// Cases where pull is required
	for _, pull := range []string{"missing", "always"} {
		t.Run("pull="+pull, func(t *testing.T) {
			base.Cmd("rmi", "-f", testutil.CommonImage).Run()
			base.Cmd("images").AssertOutNotContains(testutil.CommonImage)
			t.Cleanup(func() {
				base.ComposeCmd("-f", comp.YAMLFullPath(), "down").AssertOK()
			})
			base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "--pull", pull).AssertOutContains("hi")
		})
	}

	t.Run("pull=never, no pull", func(t *testing.T) {
		base.Cmd("rmi", "-f", testutil.CommonImage).Run()
		base.Cmd("images").AssertOutNotContains(testutil.CommonImage)
		t.Cleanup(func() {
			base.ComposeCmd("-f", comp.YAMLFullPath(), "down").AssertOK()
		})
		base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "--pull", "never").AssertExitCode(1)
	})
}

func TestComposeUpServicePullPolicy(t *testing.T) {
	base := testutil.NewBase(t)

	var dockerComposeYAML = fmt.Sprintf(`
services:
  test:
    image: %s
    command: sh -euxc "echo hi"
    pull_policy: "never"
`, testutil.CommonImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()

	base.Cmd("rmi", "-f", testutil.CommonImage).Run()
	base.Cmd("images").AssertOutNotContains(testutil.CommonImage)
	base.ComposeCmd("-f", comp.YAMLFullPath(), "up").AssertExitCode(1)
}
