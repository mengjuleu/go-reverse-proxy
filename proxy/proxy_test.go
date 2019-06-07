package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHost(t *testing.T) {
	hosts := []struct {
		url  string
		name string
		site string
	}{
		{"www.google.com",
			"",
			"",
		},
		{
			"ide.mleu.dev.devbucket.org",
			"ide",
			"mleu",
		},
		{
			"mleu.dev.devbucket.org",
			"bb",
			"mleu",
		},
		{
			"ide.mleu.devbucket.org",
			"ide",
			"mleu",
		},
		{
			"mleu.devbucket.org",
			"bb",
			"mleu",
		},
	}

	for _, host := range hosts {
		name, site := parseHost(host.url)
		require.Equal(t, name, host.name, host)
		require.Equal(t, site, host.site, host)
	}
}

// loadLadleService
func TestLoadLadleService(t *testing.T) {
	path := "./data/*.service.yaml"
	services := loadLadleService(path)

	expectedServices := []struct {
		HostName string
		Port     int
	}{
		{
			"a",
			9000,
		},
		{
			"b",
			9001,
		},
		{
			"c",
			9002,
		},
	}

	for i, s := range services {
		require.Equal(t, s.HostName, expectedServices[i].HostName, "HostName mismatch")
		require.Equal(t, s.Port, expectedServices[i].Port, "Port mismatch")
	}
}
