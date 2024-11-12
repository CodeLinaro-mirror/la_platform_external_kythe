// Copyright 2024 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type StringList []string

func (s *StringList) String() string {
	return strings.Join(*s, " ")
}

func (s *StringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

var (
	kotlinSrc       StringList
	kotlinCommonSrc StringList
	kotlinCp        StringList
	kotlinArgs      string
	kotlinOut       string

	corpus string
	vnames string

	outputKzip string
)

// TODO (spandandas): Add env
// TODO (spandandas): Add support for kotlin-home to prevent issues caused by version skew between kotlinc-jvm used by android build and kotlin extractors.
func init() {
	flag.StringVar(&outputKzip, "o", "", "Path to the output kzip file")
	flag.Var(&kotlinSrc, "srcs", "Path to .kt files passed to kotlinc. Can be passed multiple times. @ files are supported.")
	flag.Var(&kotlinCommonSrc, "common_srcs", "Path to platform agnostic .kt files passed to kotlinc. Can be passed multiple times. @ files are supported.")
	flag.Var(&kotlinCp, "cp", "Path to .jar files on the classpath for kotlinc. Can be passed multiple times. @ files are supported.")
	flag.StringVar(&kotlinArgs, "args", "", "Additional args passed to kotlinc.")
	flag.StringVar(&kotlinOut, "kotlin_out", "", "Output of kotlinc. If multiple outputs are created, pass the first one to the extractor.")
	flag.StringVar(&corpus, "corpus", "", "Corpus label to assign")
	flag.StringVar(&vnames, "vnames", "", "Path to vnames.json for custom vname mappings")
}

// Creates a kzip file in the format expected by the internal Google3 kotlin indexer
func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: kotlinc_extractor -o <output_kzip> -corpus <corpus> -args '<verbatim args to kotlinc>' [-srcs <path to .kt file>] [-cp <path to .jar file>]")
		flag.PrintDefaults()
	}

	flag.Parse()

	generator, err := initGenerator()
	maybeLogErrAndExit(err, "Could not parse args")

	err = generator.Create(outputKzip)
	maybeLogErrAndExit(err, "Could not create the kzip file: "+outputKzip)
}

func initGenerator() (*compilationUnitGenerator, error) {
	srcs, err := maybeParseRsp(kotlinSrc)
	if err != nil {
		return nil, err
	}
	commonSrcs, err := maybeParseRsp(kotlinCommonSrc)
	if err != nil {
		return nil, err
	}
	cp, err := maybeParseRsp(kotlinCp)
	if err != nil {
		return nil, err
	}

	generator := &compilationUnitGenerator{
		inputs: compilationUnitInputs{
			srcs:           srcs,
			commonSrcs:     commonSrcs,
			classpath:      cp,
			additionalArgs: strings.Split(kotlinArgs, " "),
			output:         kotlinOut,
			corpus:         corpus,
		},
	}

	if vnames != "" {
		vnameMappings, err := maybeParseVNames(vnames)
		if err != nil {
			return nil, err
		}
		generator.vnameMappings = vnameMappings
	}

	return generator, nil
}
