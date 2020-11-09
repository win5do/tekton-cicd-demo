package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
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
    name: poll-pull
spec: {}
`

	obj, err := yamlToUnstructured([]byte(pr))
	require.NoError(t, err)

	config := kConfigOrDie(false)
	config.Impersonate = rest.ImpersonationConfig{
		UserName: "system:serviceaccount:tekton-pipelines:tekton-poll-pull",
	}

	err = createPipelineRun(dynamic.NewForConfigOrDie(config), obj, 300)
	if strings.Contains(err.Error(), "admission webhook") {
		t.Logf("ignore err: %+v", err)
		err = nil
	}
	require.NoError(t, err)
}

func TestCheckExistsPipelineRun(t *testing.T) {
	pr := `
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: git-cicd-
  namespace: tekton-pipelines
  labels:
    name: poll-pull
spec: {}
`

	obj, err := yamlToUnstructured([]byte(pr))
	require.NoError(t, err)

	config := kConfigOrDie(false)

	_, err = checkExistsPipelineRun(dynamic.NewForConfigOrDie(config), obj, 300)
	require.NoError(t, err)
}
