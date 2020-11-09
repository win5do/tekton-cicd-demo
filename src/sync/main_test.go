package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReg(t *testing.T) {
	r := expImageFull.MatchString(`image: gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/controller:v0.16.3@sha256:a14129ffff1c46b39a9cb82e44096b49770efc4f1f8b85fb67d1bb00af906c44`)
	require.True(t, r)

	r = expImageFull.MatchString(`image: gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/controller:v0.16.3`)
	require.True(t, r)

	r = expImageFull.MatchString(`image: gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/controller`)
	require.True(t, r)
}
