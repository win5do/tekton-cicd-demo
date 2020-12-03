package common

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strings"
)

var (
	PrGVR = schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1beta1",
		Resource: "pipelineruns",
	}
)

func KConfigOrDie(inCluster bool) *rest.Config {
	var config *rest.Config
	var err error
	if inCluster {
		config, err = rest.InClusterConfig()

	} else {
		config, err = clientcmd.BuildConfigFromFlags("", os.Getenv(clientcmd.RecommendedConfigPathEnvVar))
	}
	if err != nil {
		log.Fatal(err)
	}

	return config
}

func InClusterNamespace() string {
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}

	return ""
}
