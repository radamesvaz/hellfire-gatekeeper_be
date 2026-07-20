package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseRefCandidates_MainFallsBackToMaster(t *testing.T) {
	assert.Equal(t, []string{"main", "origin/main", "master", "origin/master"}, baseRefCandidates("main"))
}

func TestBaseRefCandidates_MasterFallsBackToMain(t *testing.T) {
	assert.Equal(t, []string{"master", "origin/master", "main", "origin/main"}, baseRefCandidates("master"))
}

func TestBaseRefCandidates_FeatureBranch(t *testing.T) {
	assert.Equal(t, []string{"create-super-admin", "origin/create-super-admin"}, baseRefCandidates("create-super-admin"))
}
