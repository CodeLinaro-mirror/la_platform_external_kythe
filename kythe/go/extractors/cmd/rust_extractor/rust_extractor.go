// rust_extractor prepares kzip for a given Rust compilation.
// It is a "prefix tool", i.e. if Rust compiler is invoked with
//   rustc --emit link foo.rs ...
// then running
//   KYTHE_OUTPUT_FILE=foo.kzip rust_extractor rustc --emit foo.rs
// creates foo.kzip
// Kythe corpus is picked from KYTHE_CORPUS environment variable.
// KYTHE_VNAMES environment variable points the JSON file with path rewrite rules.

package main

import (
	"bufio"
	"fmt"
	"io"
	"kythe.io/kythe/go/platform/kzip"
	"kythe.io/kythe/go/util/vnameutil"
	apb "kythe.io/kythe/proto/analysis_go_proto"
	spb "kythe.io/kythe/proto/storage_go_proto"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func maybeFatal(err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}

// Context maintains extraction state and and implements extraction.
type Context struct {
	cu           *apb.CompilationUnit
	kzipPath     string
	kzipFile     *kzip.Writer
	rustCompiler string
	inputs       []string
	rewriteRules vnameutil.Rules
}

// Construct
func newContext(args []string, kzipPath string) *Context {
	ctx := &Context{
		kzipPath: kzipPath,
		cu: &apb.CompilationUnit{
			VName: &spb.VName{
				Corpus:   os.Getenv("KYTHE_CORPUS"),
				Language: "rust",
			},
		},
	}
	// Process command line arguments.
	envRex := regexp.MustCompile("^([A-Za-z_]\\w*)=(.*)$")
	for i, arg := range args {
		if m := envRex.FindStringSubmatch(arg); m != nil {
			ctx.cu.Environment = append(ctx.cu.Environment, &apb.CompilationUnit_Env{Name: m[1], Value: m[2]})
			continue
		}
		ctx.rustCompiler = arg
		ctx.cu.Argument = args[i+1:]
		break
	}
	return ctx
}

// Transforms command line arguments into our arguments.
func (ctx *Context) actualCompilerArgs() []string {
	// Drop -o and -emit arguments, prepend -emit dep-info=...,metadata=...
	args := []string{
		"--emit", fmt.Sprintf("dep-info=%s,metadata=%s", ctx.depsPath(), ctx.metadataPath()),
		"-Z", "save-analysis"}

	rdr := newArgReader(ctx.cu.Argument)
	for ; !rdr.atEnd(); rdr.read() {
		switch rdr.key() {
		case "--emit", "-o":
		default:
			args = append(args, rdr.currentArg()...)
		}
	}
	return args
}

// Runs Rust compiler, saving compilation analysis, extracts input files list
func (ctx *Context) runCompiler() {
	// Remove old files, ensure the output directory exists
	_ = os.Remove(ctx.savedAnalysisPath())
	_ = os.Remove(ctx.depsPath())
	_ = os.Remove(ctx.metadataPath())
	_ = os.Remove(ctx.kzipPath)
	err := os.MkdirAll(filepath.Dir(ctx.kzipPath), 0755)
	maybeFatal(err)

	// In addition to the actual parameters, instruct the compiler to
	// generate the dependencies file, the metadata file, and save the
	// compiler analysis as JSON file.
	cmd := exec.Command(ctx.rustCompiler, ctx.actualCompilerArgs()...)
	// Append command line environment variables
	cmd.Env = os.Environ()
	for _, x := range ctx.cu.Environment {
		cmd.Env = append(cmd.Env, x.Name+"="+x.Value)
	}
	// Append save analysis configuration, including saved analysis path
	cmd.Env = append(cmd.Env,
		fmt.Sprintf(`RUST_SAVE_ANALYSIS_CONFIG={`+
			`"output_file": %q,`+
			`"full_docs":true,`+
			`"pub_only":false,`+
			`"reachable_only":false,`+
			`"distro_crate":false,`+
			`"signatures":false,`+
			`"borrow_data":false}`, ctx.savedAnalysisPath()))

	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		fmt.Printf("%s\n", output)
	}
	maybeFatal(err)

	// Build the list of the source files
	f, err := os.Open(ctx.depsPath())
	maybeFatal(err)
	ctx.getInputFiles(f, ctx.depsPath())
	_ = f.Close()
}

// Obtains the list of the Rust source being compiled.
func (ctx *Context) getInputFiles(f io.Reader, dependentsOf string) {
	s := bufio.NewScanner(f)
	n := len(dependentsOf)
	for s.Scan() {
		ln := s.Text()
		// Look for the line:
		//  <dependentsOf>: file ...
		// Ignore not .rs files.
		if strings.HasPrefix(ln, dependentsOf) && ln[n] == ':' {
			deps := strings.Split(strings.Trim(ln[n+1:], " "), " ")
			for _, dep := range deps {
				if strings.HasSuffix(dep, ".rs") {
					ctx.inputs = append(ctx.inputs, dep)
				}
			}
		}
	}
}

// Reads vname rewriting rules
func (ctx *Context) loadRules(r io.Reader) {
	var err error
	ctx.rewriteRules, err = vnameutil.ReadRules(r)
	maybeFatal(err)
}

func (ctx *Context) makeVname(path string) *spb.VName {
	var vname *spb.VName
	var ok bool
	if vname, ok = ctx.rewriteRules.Apply(path); !ok {
		vname = &spb.VName{Path: path, Corpus: ctx.cu.VName.Corpus}
	}
	// By default, the corpus is the same as compilation unit's.
	if vname.Corpus == "" {
		vname.Corpus = ctx.cu.VName.Corpus
	}
	return vname
}

// Saves kzip file.
func (ctx *Context) saveKzip() {
	kzf, err := os.Create(ctx.kzipPath)
	maybeFatal(err)
	ctx.kzipFile, err = kzip.NewWriteCloser(kzf)
	maybeFatal(err)

	// The output archive contains an entry with save analysis
	// contents, and entry for each input file, and a proto/JSON
	// message for the compilation unit.

	// First, save analysis
	saPath := ctx.savedAnalysisPath()
	f, err := os.Open(saPath)
	maybeFatal(err)
	digest, err := ctx.kzipFile.AddFile(f)
	err = f.Close()
	maybeFatal(err)

	ctx.cu.RequiredInput = append(ctx.cu.RequiredInput, &apb.CompilationUnit_FileInput{
		VName: ctx.makeVname(saPath),
		Info:  &apb.FileInfo{Path: saPath, Digest: digest},
	})

	// Input files
	for _, input := range ctx.inputs {
		f, err = os.Open(input)
		maybeFatal(err)
		digest, err = ctx.kzipFile.AddFile(f)
		maybeFatal(err)
		err = f.Close()
		maybeFatal(err)
		vname := ctx.makeVname(input)
		vname.Language = "rust"
		ctx.cu.RequiredInput = append(ctx.cu.RequiredInput, &apb.CompilationUnit_FileInput{
			VName: vname,
			Info: &apb.FileInfo{
				Path:   input,
				Digest: digest,
			},
		})
		ctx.cu.SourceFile = append(ctx.cu.SourceFile, input)
	}

	// Finally, compilation unit
	_, err = ctx.kzipFile.AddUnit(ctx.cu, nil)
	maybeFatal(err)
	err = ctx.kzipFile.Close()
	maybeFatal(err)
}

func (ctx Context) savedAnalysisPath() string {
	return ctx.kzipPath + ".sa.json"
}

func (ctx Context) depsPath() string {
	return ctx.kzipPath + ".deps"
}

func (ctx Context) metadataPath() string {
	return ctx.kzipPath + ".metadummy"
}

// Utility stuff to iterate over rustc command-line "items"
// An item can be:
//   --foo val
//   --foo=val
//   -C val
//   -Cval
//   arg
type argReader struct {
	inArgs       []string
	currentIndex int
	nextIndex    int
	argKey       string
}

func newArgReader(args []string) *argReader {
	ret := &argReader{inArgs: args, nextIndex: 0, currentIndex: 0}
	ret.read()
	return ret
}

func (ar argReader) atEnd() bool {
	return ar.currentIndex >= len(ar.inArgs)
}

func (ar *argReader) read() {
	ar.currentIndex = ar.nextIndex
	if ar.atEnd() {
		return
	}
	ar.nextIndex++
	ar.argKey = ar.inArgs[ar.currentIndex]
	if ar.argKey[0] != '-' {
		return
	}
	n := -1
	if strings.HasPrefix(ar.argKey, "--") {
		n = strings.Index(ar.argKey, "=")
	} else if len(ar.argKey) > 2 {
		n = 2
	}
	if n >= 0 {
		ar.argKey = ar.argKey[0:n]
	} else if !ar.atEnd() {
		ar.nextIndex++
	}
}

func (ar *argReader) currentArg() []string {
	return ar.inArgs[ar.currentIndex:ar.nextIndex]
}

func (ar argReader) key() string {
	return ar.argKey
}

func main() {
	if len(os.Args) < 2 {
		maybeFatal(fmt.Errorf("at least the rust compiler path should be present"))
	}
	const KzipEnv = "KYTHE_OUTPUT_FILE"
	kzipPath := os.Getenv(KzipEnv)
	if kzipPath == "" {
		maybeFatal(fmt.Errorf("%s is  not set", KzipEnv))
	}
	ctx := newContext(os.Args[1:], kzipPath)
	vnamesPath := os.Getenv("KYTHE_VNAMES")
	if vnamesPath != "" {
		vf, err := os.Open(vnamesPath)
		maybeFatal(err)
		ctx.loadRules(vf)
	}
	ctx.runCompiler()
	ctx.saveKzip()
}
