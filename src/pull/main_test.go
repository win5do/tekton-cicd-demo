package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"strings"
	"tekton/utils/common"
	"testing"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestGitNewestCommit(t *testing.T) {
	commit, err := gitNewestCommit(
		"https://github.com/win5do/tekton-cicd-demo.git",
		"",
		"",
		"*",
		10*60,
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
    name: git-polling
spec: {}
`

	obj, err := yamlToUnstructured([]byte(pr))
	require.NoError(t, err)

	config := common.KConfigOrDie(false)
	config.Impersonate = rest.ImpersonationConfig{
		UserName: "system:serviceaccount:tekton-pipelines:tekton-utils",
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
    name: git-polling
spec: {}
`

	obj, err := yamlToUnstructured([]byte(pr))
	require.NoError(t, err)

	config := common.KConfigOrDie(false)

	_, err = checkExistsPipelineRun(dynamic.NewForConfigOrDie(config), obj, 300)
	require.NoError(t, err)
}
