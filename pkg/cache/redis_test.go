package cache_test

import (
	"context"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/mediocregopher/radix/v3"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/xakep666/licensevalidator/pkg/cache"
	"github.com/xakep666/licensevalidator/pkg/validation"
)

type RedisCacheTestSuite struct {
	suite.Suite

	licenseResolverMock *validation.LicenseResolverMock
	redisContainer      testcontainers.Container
	redisClient         *radix.Pool
	cache               *cache.RedisCache
}

func (s *RedisCacheTestSuite) TestResolveLicense() {
	module := validation.Module{
		Name:    "test-name",
		Version: semver.MustParse("v1.0.0"),
	}

	license := validation.License{
		Name:   "MIT License",
		SPDXID: "MIT",
	}

	s.licenseResolverMock.On("ResolveLicense", mock.Anything, module).Return(license, nil).Once()

	actualLicense, err := s.cache.ResolveLicense(context.Background(), module)
	if s.NoError(err) {
		s.Equal(license, actualLicense)
	}

	// 2nd call should be in cache
	actualLicense, err = s.cache.ResolveLicense(context.Background(), module)
	if s.NoError(err) {
		s.Equal(license, actualLicense)
	}
}

func (s *RedisCacheTestSuite) SetupSuite() {
	var err error
	s.redisContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ProviderType: testcontainers.ProviderDocker,
		Started:      true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:6-alpine",
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
			ExposedPorts: []string{"6379/tcp"},
		},
	})
	s.Require().NoError(err)

	s.redisContainer.FollowOutput(LogConsumerFunc(func(log testcontainers.Log) {
		s.T().Logf("redis [%s]: %s", log.LogType, log.Content)
	}))
	s.Require().NoError(s.redisContainer.StartLogProducer(context.Background()))

	redisEp, err := s.redisContainer.PortEndpoint(context.Background(), "6379/tcp", "")
	s.Require().NoError(err)

	s.redisClient, err = radix.NewPool("tcp", redisEp, 10)
	s.Require().NoError(err)
}

func (s *RedisCacheTestSuite) SetupTest() {
	s.licenseResolverMock = new(validation.LicenseResolverMock)

	s.cache = &cache.RedisCache{
		Backed: cache.Direct{
			LicenseResolver: s.licenseResolverMock,
		},
		Client: s.redisClient,
	}
}

func (s *RedisCacheTestSuite) TearDownTest() {
	s.licenseResolverMock.AssertExpectations(s.T())
	s.Require().NoError(s.redisClient.Do(radix.Cmd(nil, "FLUSHALL")))
}

func (s *RedisCacheTestSuite) TearDownSuite() {
	s.Require().NoError(s.redisContainer.Terminate(context.Background()))
}

func TestRedisCache_Suite(t *testing.T) {
	if testing.Short() {
		t.Skipf("Skipping integration test in short mode")
		return
	}

	suite.Run(t, new(RedisCacheTestSuite))
}

type LogConsumerFunc func(log testcontainers.Log)

func (f LogConsumerFunc) Accept(log testcontainers.Log) { f(log) }
