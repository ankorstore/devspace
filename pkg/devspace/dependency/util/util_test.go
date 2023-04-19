package util

import (
	"testing"

	"gotest.tools/assert"
)

func TestSwitchURLType(t *testing.T) {
	httpURL := "https://github.com/ankorstore/devspace.git"
	sshURL := "git@github.com:ankorstore/devspace.git"

	assert.Equal(t, sshURL, switchURLType(httpURL))
	assert.Equal(t, httpURL, switchURLType(sshURL))
}
