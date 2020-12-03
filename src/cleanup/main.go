package main

import (
	"context"
	"time"

	errors2 "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"

	"tekton/utils/common"
)

var (
	flagRange            int
	flagExcludedSelector string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:  "main",
		Long: `cleanup old PipelineRun with cronJob`,
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

	rootCmd.Flags().IntVar(&flagRange, "range", 259200, "clean PipelineRun created before this number of hours") // 3d
	rootCmd.Flags().StringVar(&flagExcludedSelector, "excluded-selector", "", "excluded label selector that won't be cleanup")

	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	set, err := labels.ConvertSelectorToLabelsMap(flagExcludedSelector)
	if err != nil {
		return errors2.WithStack(err)
	}

	return cleanup(dynamic.NewForConfigOrDie(common.KConfigOrDie(true)), flagRange, set)
}

func cleanup(client dynamic.Interface, timeRange int, excludedLabels labels.Set) error {
	log.Debugf("excludedLabels: %+v", excludedLabels)

	namespace := common.InClusterNamespace()

	log.Debugf("inClusterNamespace: %+v", namespace)

	r, err := client.Resource(common.PrGVR).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return errors2.WithStack(err)
	}

	if len(r.Items) == 0 {
		return nil
	}

	for _, v := range r.Items {
		if v.GetCreationTimestamp().After(time.Now().Add(-time.Duration(timeRange) * time.Second)) {
			// new create
			continue
		}

		if labels.SelectorFromSet(excludedLabels).Matches(labels.Set(v.GetLabels())) {
			continue
		}

		err := client.Resource(common.PrGVR).Namespace(namespace).Delete(context.Background(), v.GetName(), metav1.DeleteOptions{})
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return errors2.WithStack(err)
			}
		}
		log.Debugf("clean: %s", v.GetName())
	}

	return nil
}
