package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/rancher/agent/cluster"
	"github.com/rancher/agent/node"
	"github.com/rancher/rancher/pkg/remotedialer"
	"github.com/sirupsen/logrus"
)

const (
	Token  = "X-API-Tunnel-Token"
	Params = "X-API-Tunnel-Params"
)

func main() {
	if os.Getenv("CATTLE_DEBUG") == "true" || os.Getenv("RANCHER_DEBUG") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func getParams() (map[string]interface{}, error) {
	if os.Getenv("CATTLE_CLUSTER") == "true" {
		return cluster.Params()
	}
	return node.Params(), nil
}

func getTokenAndURL() (string, string, error) {
	if os.Getenv("CATTLE_CLUSTER") == "true" {
		return cluster.TokenAndURL()
	}
	return node.TokenAndURL()
}

func run() error {
	params, err := getParams()
	if err != nil {
		return err
	}

	bytes, err := json.Marshal(params)
	if err != nil {
		return err
	}

	token, server, err := getTokenAndURL()
	if err != nil {
		return err
	}

	headers := map[string][]string{
		Token:  {token},
		Params: {base64.StdEncoding.EncodeToString(bytes)},
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return err
	}

	wsURL := fmt.Sprintf("wss://%s/v3/connect", serverURL.Host)
	logrus.Infof("Connecting to %s with token %s", wsURL, token)
	remotedialer.ClientConnect(wsURL, http.Header(headers), nil, func(proto, address string) bool {
		switch proto {
		case "tcp":
			return true
		case "unix":
			return address == "/var/run/docker.sock"
		}
		return false
	})

	return nil
}
