package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strings"
	"testing"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestGitNewestCommit(t *testing.T) {
	commit, err := gitNewestCommit(
		"https://github.com/win5do/tekton-cicd-demo.git",
		"xxx",
		"xxx",
		"*",
		3000,
	)
	require.NoError(t, err)
	if commit != nil {
		t.Logf("%s", commit.Hash)
	}
}

func TestTemplate(t *testing.T) {
	tpl, err := parseTemplate("testdata/tpl.txt")
	require.NoError(t, err)
	r, err := applyTemplate(tpl, map[string]string{
		"REPO":   "http://xxx.com/yyy/zzz.git",
		"COMMIT": "abc123xyz",
	})
	require.NoError(t, err)
	t.Logf("%s", r)
}

func TestCreatePipelineRun(t *testing.T) {
	pr := `
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: git-cicd-
  namespace: tekton-pipelines
  labels:
    name: git-pull
spec:
`

	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv(clientcmd.RecommendedConfigPathEnvVar))
	require.NoError(t, err)
	config.Impersonate = rest.ImpersonationConfig{
		UserName: "system:serviceaccount:tekton-pipelines:tekton-poll-pull",
	}
	err = createPipelineRun(config, []byte(pr))
	if strings.Contains(err.Error(), "admission webhook") {
		t.Logf("ignore err: %+v", err)
		err = nil
	}
	require.NoError(t, err)
}
