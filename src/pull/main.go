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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
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

	rootCmd.Execute()
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
		"REPO":      flagRepo,
		"SHA":       commit.Hash.String(),
		"SHORT_SHA": commit.Hash.String()[:7],
	})
	if err != nil {
		return errors2.WithStack(err)
	}

	err = createPipelineRun(nil, pr)
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

func applyTemplate(t *template.Template, params map[string]string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := t.Execute(buf, params)
	if err != nil {
		return nil, errors2.WithStack(err)
	}
	return buf.Bytes(), nil
}

func createPipelineRun(config *rest.Config, pr []byte) error {
	m := make(map[string]interface{})
	err := yaml.Unmarshal(pr, &m)
	if err != nil {
		return errors2.WithStack(err)
	}

	obj := &unstructured.Unstructured{
		Object: m,
	}

	if config == nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return errors2.WithStack(err)
		}
	}
	client := dynamic.NewForConfigOrDie(config)

	log.Debugf("obj: %+v", obj)

	r, err := client.Resource(schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1beta1",
		Resource: "pipelineruns",
	}).Namespace(obj.GetNamespace()).Create(context.Background(), obj, metav1.CreateOptions{})
	if err != nil {
		return errors2.WithStack(err)
	}

	log.Infof("PipelineRun created success, name: %s", r.GetName())
	return nil
}
