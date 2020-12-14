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

func TestMatchRerunGenerateName(t *testing.T) {
	r := matchRerunGenerateName("foo-bar-r-dcfht")
	require.True(t, r)

	r = matchRerunGenerateName("foo-bar-6cfh0")
	require.False(t, r)

	r = matchRerunGenerateName("foo-bar-6cfht")
	require.False(t, r)

	r = matchRerunGenerateName("foo-r-bar")
	require.False(t, r)
}
