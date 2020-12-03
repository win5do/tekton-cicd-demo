package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/labels"
)

func TestLabelsMatch(t *testing.T) {
	exclude := labels.Set{
		"type": "origin",
	}

	in := labels.Set{
		"type": "origin",
		"app":  "golang",
	}

	r := labels.SelectorFromSet(exclude).Matches(in)
	require.True(t, r)
}
