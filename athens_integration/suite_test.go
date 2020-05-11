package athens_integration_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/xakep666/licensevalidator/internal/testutil"
)

var (
	_, b, _, _     = runtime.Caller(0)
	basepath       = filepath.Dir(b)
	projectPath, _ = filepath.Abs(filepath.Join(basepath, ".."))
)

type AthensIntegrationTestSuite struct {
	suite.Suite

	network         testcontainers.Network
	appContainer    testcontainers.Container
	athensContainer testcontainers.Container
	httpClient      *http.Client
}

func (s *AthensIntegrationTestSuite) proxyGet(path string) *http.Response {
	s.T().Helper()
	ep, err := s.athensContainer.PortEndpoint(context.Background(), "3000/tcp", "http")
	s.Require().NoError(err)

	resp, err := s.httpClient.Get(ep + path)
	s.Require().NoError(err)

	s.T().Cleanup(func() {
		resp.Body.Close()
	})

	return resp
}

func (s *AthensIntegrationTestSuite) proxyGetZip(module, version string) *http.Response {
	s.T().Helper()
	return s.proxyGet(fmt.Sprintf("/%s/@v/%s.zip", module, version))
}

func (s *AthensIntegrationTestSuite) proxyGetMod(module, version string) *http.Response {
	s.T().Helper()
	return s.proxyGet(fmt.Sprintf("/%s/@v/%s.mod", module, version))
}

func (s *AthensIntegrationTestSuite) proxyListVersions(module string) *http.Response {
	s.T().Helper()
	return s.proxyGet(fmt.Sprintf("/%s/@v/list", module))
}

func (s *AthensIntegrationTestSuite) SetupSuite() {
	dp, err := testcontainers.NewDockerProvider() // for networking
	s.Require().NoError(err)

	netName := fmt.Sprintf("licensevalidator-test-net-%d", time.Now().UnixNano())

	s.network, err = testcontainers.GenericNetwork(context.Background(), testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name:       netName,
			Attachable: true,
		},
		ProviderType: testcontainers.ProviderDocker,
	})
	s.Require().NoError(err)

	s.appContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    projectPath,
				Dockerfile: filepath.Join("cmd", "licensevalidator", "Dockerfile"),
			},
			ExposedPorts: []string{"8080/tcp"},
			WaitingFor: wait.ForHTTP("/").
				WithPort("8080/tcp").
				WithStatusCodeMatcher(func(status int) bool {
					return status < http.StatusInternalServerError
				}),
			BindMounts: map[string]string{
				filepath.Join(basepath, "testdata", "licensevalidator_cfg.toml"): filepath.FromSlash("/etc/licensevalidator.toml"),
			},
			AutoRemove:  true,
			NetworkMode: container.NetworkMode(netName),
		},
		Started:      true,
		ProviderType: testcontainers.ProviderDocker,
	})
	s.Require().NoError(err)
	s.appContainer.FollowOutput(&testutil.TLogConsumer{
		T:      s.T(),
		Prefix: "LicenseValidator",
	})
	s.Require().NoError(s.appContainer.StartLogProducer(context.Background()))

	netObject, err := dp.GetNetwork(context.Background(), testcontainers.NetworkRequest{
		Name: netName,
	})
	s.Require().NoError(err)

	appIP, _, err := net.ParseCIDR(netObject.Containers[s.appContainer.GetContainerID()].IPv4Address)
	s.Require().NoError(err)

	s.athensContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "gomods/athens:v0.7.0",
			ExposedPorts: []string{"3000/tcp"},
			WaitingFor:   wait.ForHTTP("/").WithPort("3000/tcp"),
			AutoRemove:   true,
			BindMounts: map[string]string{
				filepath.Join(basepath, "testdata", "athens_cfg.toml"): filepath.FromSlash("/config/config.toml"),
			},
			Env: map[string]string{
				"ATHENS_PROXY_VALIDATOR": fmt.Sprintf("http://%s:8080/athens/admission", appIP),
			},
			NetworkMode: container.NetworkMode(netName),
		},
		Started:      true,
		ProviderType: testcontainers.ProviderDocker,
	})
	s.Require().NoError(err, "athens startup failed")

	s.athensContainer.FollowOutput(&testutil.TLogConsumer{
		T:      s.T(),
		Prefix: "Athens",
	})
	s.Require().NoError(s.athensContainer.StartLogProducer(context.Background()))

	s.httpClient = &http.Client{}
}

func (s *AthensIntegrationTestSuite) TearDownSuite() {
	athensStopCtx, athensStopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer athensStopCancel()
	s.NoError(s.athensContainer.Terminate(athensStopCtx))

	appStopCtx, appStopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer appStopCancel()
	s.NoError(s.appContainer.Terminate(appStopCtx))

	s.NoError(s.network.Remove(context.Background()))
}

func TestAthensIntegration_suite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
		return
	}

	suite.Run(t, new(AthensIntegrationTestSuite))
}
