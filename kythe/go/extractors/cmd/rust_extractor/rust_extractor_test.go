package main

import (
	"kythe.io/kythe/go/test/testutil"
	apb "kythe.io/kythe/proto/analysis_go_proto"
	spb "kythe.io/kythe/proto/storage_go_proto"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestNewContext(t *testing.T) {
	t.Run("processCommandLine", func(t *testing.T) {
		line := "FOO=x RUST_VERSION=1.59 compiler arg"
		ctx := newContext(strings.Split(line, " "), "foo.kzip")
		if err := testutil.DeepEqual(
			[]*apb.CompilationUnit_Env{
				{Name: "FOO", Value: "x"},
				{Name: "RUST_VERSION", Value: "1.59"}},
			ctx.cu.Environment); err != nil {
			t.Errorf("wrong environment: %s", err)
		}
		if err := testutil.DeepEqual("compiler", ctx.rustCompiler); err != nil {
			t.Errorf("wrong compiler: %s", err)
		}
		if err := testutil.DeepEqual([]string{"arg1"}, ctx.cu.Argument); err == nil {
			t.Errorf("wrong arguments: %s", err)
		}

	})
}

func TestContext_compilerArgs(t *testing.T) {
	cmdLine := "rustc -C linker=lld --emit link -o foo lib.rs --extern std=bar.so -Zremap-cwd-prefix= --crate-type=rlib"
	expected := "--emit dep-info=foo.kzip.deps,metadata=foo.kzip.metadummy -Z save-analysis " +
		"-C linker=lld lib.rs --extern std=bar.so -Zremap-cwd-prefix= --crate-type=rlib"

	t.Run("parseArgs1", func(t *testing.T) {
		ctx := newContext(strings.Split(cmdLine, " "), "foo.kzip")
		if err := testutil.DeepEqual(strings.Split(expected, " "), ctx.actualCompilerArgs()); err != nil {
			t.Errorf("wrong compiler arguments: %s", err)
		}
	})
}

func TestContext_getInputFiles(t *testing.T) {
	t.Run("input files", func(t *testing.T) {
		ctx := newContext(strings.Split("rustc foo.rs", " '"), "foo.kzip")
		deps := `
foo.rmeta: x.rs x.der
foo.deps: foo.rs y.der
`
		ctx.getInputFiles(strings.NewReader(deps), "foo.deps")
		if err := testutil.DeepEqual([]string{"foo.rs"}, ctx.inputs); err != nil {
			t.Errorf("wrong input files: %s", err)
		}
	})
}

func TestContext_makeVname(t *testing.T) {
	t.Run("name rewriting", func(t *testing.T) {
		vnameJson := `[
  {
    "pattern": "out/(.*)",
    "vname": {
      "root": "out",
      "path": "@1@"
    }
  },
  {
    "pattern": "(.*)",
    "vname": {
      "path": "@1@"
    }
  }
]
`
		tests := []struct {
			path string
			want *spb.VName
		}{
			// static
			{"foo/bar.rs", &spb.VName{Path: "foo/bar.rs", Corpus: "aosp"}},
			{"out/path.json", &spb.VName{Root: "out", Path: "path.json", Corpus: "aosp"}},
		}

		_ = os.Setenv("KYTHE_CORPUS", "aosp")
		ctx := newContext(strings.Split("rustc foo.rs", " '"), "foo.kzip")
		ctx.loadRules(strings.NewReader(vnameJson))
		for _, test := range tests {
			if got := ctx.makeVname(test.path); !reflect.DeepEqual(got, test.want) {
				t.Errorf("makeVname() = %v, want %v", got, test.want)
			}
		}
	})
}
