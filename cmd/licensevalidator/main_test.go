package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestAppRunsWithSample(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skipf("Skip integration test in short mode")
		return
	}

	la := ConfigSample.Server.ListenAddr

	// firstly ensure that addr from config sample can be listened
	listener, err := net.Listen("tcp", la)
	if err != nil {
		t.Skipf("Can't run test because port from sample is not available: %s", err)
		return
	}

	require.NoError(t, listener.Close())

	cli.OsExiter = func(code int) {
		t.Helper()

		if code != 0 {
			t.Fatalf("App exit with code: %d", code)
		}
		return
	}

	t.Cleanup(func() { cli.OsExiter = os.Exit })

	// get a config sample
	var configSample bytes.Buffer
	configSampleOut = &configSample
	t.Cleanup(func() { configSampleOut = os.Stdout })
	args = append(os.Args[:1], "sample-config")
	main()

	t.Logf("Got config sample:\n%s", configSample.Bytes())

	// drop this sample to file
	tmpConfig, err := ioutil.TempFile("", "licensevalidator-cfg")
	require.NoError(t, err)

	_, err = configSample.WriteTo(tmpConfig)
	require.NoError(t, err)

	t.Cleanup(func() { os.Remove(tmpConfig.Name()) })

	tmpConfig.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	// here goes app
	go func() {
		defer wg.Done()
		args = append(os.Args[:1], "-c", tmpConfig.Name())
		main()
	}()

	// here goes probe testing
	go func() {
		t.Helper()

		defer wg.Done()

		_, port, err := net.SplitHostPort(la)
		assert.NoError(t, err)

		assert.Eventually(t, func() bool {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%s", port))
			if err != nil {
				t.Logf("Probe error: %s", err)
				return false
			}
			defer resp.Body.Close()

			t.Logf("Probe response code is %d", resp.StatusCode)
			return resp.StatusCode < http.StatusInternalServerError
		}, 5*time.Minute, 10*time.Second, "server didn't become ready")

		// send sigterm to stop app
		interruptChan <- syscall.SIGTERM
	}()

	wg.Wait()
}
