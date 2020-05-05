package athens_integration_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
)

func (s *AthensIntegrationTestSuite) TestReceiveMod() {
	resp := s.proxyGetMod("github.com/stretchr/testify", "v1.5.1")

	m, err := ioutil.ReadAll(resp.Body)
	s.Require().NoError(err)

	const testifyMod = `module github.com/stretchr/testify

require (
	github.com/davecgh/go-spew v1.1.0
	github.com/pmezard/go-difflib v1.0.0
	github.com/stretchr/objx v0.1.0
	gopkg.in/yaml.v2 v2.2.2
)

go 1.13
`

	if s.Equal(http.StatusOK, resp.StatusCode) {
		s.Equal(testifyMod, string(m))
	}
}

func (s *AthensIntegrationTestSuite) TestReceiveZip() {
	resp := s.proxyGetZip("github.com/stretchr/testify", "v1.5.1")

	// hash of zip archive
	const sha256sum = "b0d1f439dfc42208b7c120dfdecee61c697496c1688361aeb100b01366d945f7"

	hash := sha256.New()
	_, err := io.Copy(hash, resp.Body)
	s.Require().NoError(err)

	if s.Equal(http.StatusOK, resp.StatusCode) {
		s.Equal(sha256sum, hex.EncodeToString(hash.Sum(nil)), "checksums are different, check athens storage health")
	}
}

func (s *AthensIntegrationTestSuite) TestReceiveForbiddenMod() {
	// agpl-3 licensed project, license forbidden in config
	resp := s.proxyGetMod("github.com/kaaryasthan/kaaryasthan", "v0.0.0-20200212235836-974506c24abc")
	s.Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *AthensIntegrationTestSuite) TestReceiveForbiddenZip() {
	resp := s.proxyGetZip("github.com/kaaryasthan/kaaryasthan", "v0.0.0-20200212235836-974506c24abc")
	s.Equal(http.StatusForbidden, resp.StatusCode)
}
