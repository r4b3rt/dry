package docker

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/docker/cli/opts"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/sockets"
	homedir "github.com/mitchellh/go-homedir"
	drytls "github.com/moncho/dry/tls"
	"github.com/moncho/dry/version"
)

const (
	//DefaultConnectionTimeout is the timeout for connecting with the Docker daemon
	DefaultConnectionTimeout = 32 * time.Second
)

var defaultDockerPath string

var headers = map[string]string{
	"User-Agent": "dry/" + version.VERSION,
}

func init() {
	defaultDockerPath, _ = homedir.Expand("~/.docker")
}
func connect(client client.APIClient, env Env) (*Daemon, error) {
	store, err := NewDockerContainerStore(client)
	if err != nil {
		return nil, err
	}
	d := &Daemon{
		client:    client,
		err:       err,
		s:         store,
		dockerEnv: env,
		resolver:  newResolver(client, false),
	}
	if err := d.init(); err != nil {
		return nil, err
	}
	return d, nil
}

func getServerHost(env Env) (string, error) {

	host := env.DockerHost
	if host == "" {
		host = DefaultDockerHost
	}

	return opts.ParseHost(env.DockerCertPath != "", host)
}

func newHTTPClient(host string, config *tls.Config) (*http.Client, error) {
	if config == nil {
		// let the api client configure the default transport.
		return nil, nil
	}

	url, err := client.ParseHostURL(host)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		TLSClientConfig: config,
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(url.Scheme, url.Host, DefaultConnectionTimeout)
		},
	}

	if err = sockets.ConfigureTransport(transport, url.Scheme, url.Host); err != nil {
		return nil, err
	}

	return &http.Client{
		Transport:     transport,
		CheckRedirect: client.CheckRedirect,
	}, nil
}

//ConnectToDaemon connects to a Docker daemon using the given properties.
func ConnectToDaemon(env Env) (*Daemon, error) {

	host, err := getServerHost(env)
	if err != nil {
		return nil, fmt.Errorf("invalid Host: %v", err)
	}
	var tlsConfig *tls.Config
	//If a path to certificates is given use the path to read certificates from
	if dockerCertPath := env.DockerCertPath; dockerCertPath != "" {
		options := drytls.Options{
			CAFile:             filepath.Join(dockerCertPath, "ca.pem"),
			CertFile:           filepath.Join(dockerCertPath, "cert.pem"),
			KeyFile:            filepath.Join(dockerCertPath, "key.pem"),
			InsecureSkipVerify: env.DockerTLSVerify,
		}
		tlsConfig, err = drytls.Client(options)
		if err != nil {
			return nil, fmt.Errorf("TLS setup error: %v", err)
		}
	} else if env.DockerTLSVerify {
		//No cert path is given but TLS verify is set, default location for
		//docker certs will be used.
		//See https://docs.docker.com/engine/security/https/#secure-by-default
		//Fixes #23
		options := drytls.Options{
			CAFile:             filepath.Join(defaultDockerPath, "ca.pem"),
			CertFile:           filepath.Join(defaultDockerPath, "cert.pem"),
			KeyFile:            filepath.Join(defaultDockerPath, "key.pem"),
			InsecureSkipVerify: env.DockerTLSVerify,
		}
		env.DockerCertPath = defaultDockerPath
		tlsConfig, err = drytls.Client(options)
		if err != nil {
			return nil, fmt.Errorf("TLS setup error: %w", err)
		}
	}
	httpClient, err := newHTTPClient(host, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("httpClient creation error: %w", err)
	}

	client, err := client.NewClient(host, env.DockerAPIVersion, httpClient, headers)
	if err == nil {
		return connect(client, env)
	}
	return nil, fmt.Errorf("error creating client: %w", err)
}
