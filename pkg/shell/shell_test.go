package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUsableShell(t *testing.T) {
	c, err := GetUsableShell()
	if !assert.NoError(t, err) {
		return
	}
	err = c.Start()
	if !assert.NoError(t, err) {
		return
	}
	c.Process.Kill()
}
