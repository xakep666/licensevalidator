// Package athens_integration contains integration tests with Athens proxy server.
// This package uses testcontainers-go library to manage docker containers (with app and Athens) and networks.
// To run this tests Docker should be installed.
// It's recommended to run tests on host but container with host network and forwarded socket also works.
package athens_integration
