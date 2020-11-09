package main

import (
	"bytes"
	"context"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	errors2 "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"sigs.k8s.io/yaml"
	"text/template"
	"time"
)

var (
	flagRepo     string
	flagUsername string
	flagPassword string
	flagBranch   string
	flagRange    int
	flagTemplate string
)

const (
	allBranch = "*"

	REPO = "REPO"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:  "main",
		Long: `git poll pull work with cronJob`,
		Run: func(cmd *cobra.Command, args []string) {
			log.SetLevel(log.DebugLevel)
			log.SetReportCaller(true)
			err := run()
			if err != nil {
				log.Debugf("err: %+v", err)
				return
			}
		},
	}
	rootCmd.Flags().StringVar(&flagRepo, "repo", "", "git clone http url")
	rootCmd.Flags().StringVar(&flagUsername, "username", "", "git username")
	rootCmd.Flags().StringVar(&flagPassword, "password", "", "git password")
	rootCmd.Flags().StringVar(&flagBranch, "branch", allBranch, "fetch branch")
	rootCmd.Flags().IntVar(&flagRange, "range", 120, "fetch time range, unit: seconds")
	rootCmd.Flags().StringVar(&flagTemplate, "template", "", "PipelineRun template")

	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// validate template first
	tpl, err := parseTemplate(flagTemplate)
	if err != nil {
		return errors2.WithStack(err)
	}

	commit, err := gitNewestCommit(flagRepo, flagUsername, flagPassword, flagBranch, flagRange)
	if err != nil {
		return errors2.WithStack(err)
	}
	if commit == nil {
		log.Infof("no new commit")
		return nil
	}

	log.Infof("newest commit: %s", commit)

	pr, err := applyTemplate(tpl, map[string]string{
		REPO:        flagRepo,
		"SHA":       commit.Hash.String(),
		"SHORT_SHA": commit.Hash.String()[:7],
	})
	if err != nil {
		return errors2.WithStack(err)
	}

	err = createPipelineRun(dynamic.NewForConfigOrDie(kConfigOrDie(true)), pr, flagRange)
	if err != nil {
		return errors2.WithStack(err)
	}

	return nil
}

func gitCloneRepo(repo, username, password string) (*git.Repository, error) {
	dir, err := ioutil.TempDir(os.TempDir(), "tekton-pull-")
	if err != nil {
		return nil, errors2.WithStack(err)
	}
	log.Infof("git clone dir: %s", dir)
	r, err := git.PlainClone(dir, true, &git.CloneOptions{
		URL: repo,
		Auth: &http.BasicAuth{
			Username: username,
			Password: password,
		},
	})
	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			r, err = git.PlainOpen(dir)
			if err != nil {
				return nil, errors2.WithStack(err)
			}
		} else {
			return nil, errors2.WithStack(err)
		}
	}

	return r, nil
}

func gitNewestCommit(repo, username, password, branch string, range_ int) (*object.Commit, error) {
	r, err := gitCloneRepo(repo, username, password)
	if err != nil {
		return nil, errors2.WithStack(err)
	}
	// retrieves the commit history
	until := time.Now()
	since := until.Add(-time.Duration(range_) * time.Second)
	logOpt := &git.LogOptions{
		Since: &since,
		Until: &until,
		All:   false,
	}

	if branch == allBranch {
		logOpt.All = true
	} else {
		ref, err := r.Reference(plumbing.NewRemoteReferenceName("origin", branch), true)
		if err != nil {
			return nil, errors2.WithStack(err)
		}

		logOpt.From = ref.Hash()
	}

	cIter, err := r.Log(logOpt)
	if err != nil {
		return nil, errors2.WithStack(err)
	}

	newestCommit, err := cIter.Next()
	if err != nil {
		if err == io.EOF {
			// no new commit
			return nil, nil
		}
		return nil, errors2.WithStack(err)
	}

	return newestCommit, nil
}

func parseTemplate(path string) (*template.Template, error) {
	t, err := template.ParseFiles(path)
	if err != nil {
		return nil, errors2.WithStack(err)
	}

	return t, nil
}

func applyTemplate(t *template.Template, params map[string]string) (*unstructured.Unstructured, error) {
	buf := bytes.NewBuffer(nil)
	err := t.Execute(buf, params)
	if err != nil {
		return nil, errors2.WithStack(err)
	}

	obj, err := yamlToUnstructured(buf.Bytes())
	if err != nil {
		return nil, errors2.WithStack(err)
	}

	delete(params, REPO) // url is invalid label value
	obj.SetLabels(labels.Merge(obj.GetLabels(), params))

	return obj, nil
}

var (
	prGVR = schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1beta1",
		Resource: "pipelineruns",
	}
)

func createPipelineRun(client dynamic.Interface, obj *unstructured.Unstructured, timeRange int) error {
	log.Debugf("obj: %+v", obj)

	var err error

	exists, err := checkExistsPipelineRun(client, obj, timeRange)
	if err != nil {
		return errors2.WithStack(err)
	}
	if exists {
		return nil
	}

	r, err := client.Resource(prGVR).Namespace(obj.GetNamespace()).Create(context.Background(), obj, metav1.CreateOptions{})
	if err != nil {
		return errors2.WithStack(err)
	}

	log.Infof("PipelineRun created success, name: %s", r.GetName())
	return nil
}

func checkExistsPipelineRun(client dynamic.Interface, obj *unstructured.Unstructured, timeRange int) (bool, error) {
	r, err := client.Resource(prGVR).Namespace(obj.GetNamespace()).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.FormatLabels(obj.GetLabels()),
	})
	if err != nil {
		return false, errors2.WithStack(err)
	}

	if len(r.Items) == 0 {
		return false, nil
	}

	if r.Items[0].GetCreationTimestamp().After(time.Now().Add(-time.Duration(timeRange) * time.Second)) {
		// create recently
		return true, nil
	}

	return false, nil
}

func yamlToUnstructured(y []byte) (*unstructured.Unstructured, error) {
	m := make(map[string]interface{})
	err := yaml.Unmarshal(y, &m)
	if err != nil {
		return nil, errors2.WithStack(err)
	}

	return &unstructured.Unstructured{
		Object: m,
	}, nil
}

func kConfigOrDie(inCluster bool) *rest.Config {
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
