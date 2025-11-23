package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/require"
	exec "golang.org/x/sys/execabs"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"allowtags": main,
	})
}

func TestScripts(t *testing.T) {
	t.Parallel()

	var goEnv struct {
		GOCACHE    string
		GOMODCACHE string
		GOMOD      string
	}

	out, err := exec.Command("go", "env", "-json").CombinedOutput()
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(out, &goEnv))

	p := testscript.Params{
		Dir: filepath.Join("testdata", "script"),
		Setup: func(env *testscript.Env) error {
			env.Setenv("GOCACHE", goEnv.GOCACHE)
			env.Setenv("GOMODCACHE", goEnv.GOMODCACHE)
			env.Setenv("GOMOD_DIR", filepath.Dir(goEnv.GOMOD))

			return nil
		},
	}

	require.NoError(t, gotooltest.Setup(&p))

	testscript.Run(t, p)
}
