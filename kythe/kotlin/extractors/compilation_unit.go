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
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"kythe.io/kythe/go/platform/kzip"
	agp "kythe.io/kythe/proto/analysis_go_proto"
	sgp "kythe.io/kythe/proto/storage_go_proto"
)

type compilationUnitInputs struct {
	srcs           StringList
	commonSrcs     StringList
	classpath      StringList
	additionalArgs StringList

	output string // Output .jar containing the compiled classes

	corpus string
}

type VNameMapping struct {
	Pattern string    `json:"pattern"`
	VName   sgp.VName `json:"vname"`
}

// compilationUnitGenerator creates the .kzip file from compilationUnitInputs
type compilationUnitGenerator struct {
	inputs        compilationUnitInputs
	vnameMappings []VNameMapping
}

func (c *compilationUnitGenerator) Create(outputKzip string) (err error) {
	var kzipFile *kzip.Writer
	defer func() {
		// close the kzip file
		// run it using defer to ensure that the file is always closed.
		err = kzipFile.Close()
	}()

	kzipFile, err = c.openKzip(outputKzip)
	if err != nil {
		return
	}
	err = c.writeCompilationUnit(kzipFile)
	return
}

func (c *compilationUnitGenerator) openKzip(outputKzip string) (*kzip.Writer, error) {
	f, err := os.Create(outputKzip)
	if err != nil {
		return nil, err
	}
	return kzip.NewWriteCloser(f)
}

func (c *compilationUnitGenerator) writeCompilationUnit(kzipFile *kzip.Writer) error {
	if c.inputs.srcs == nil {
		return errors.New("Cannot create compilation unit. Srcs not found")
	}

	cu := &agp.CompilationUnit{
		VName: &sgp.VName{
			Corpus:   c.inputs.corpus,
			Language: "kotlin",
		},
		Argument: c.getKotlincArgs(),
	}
	// Add the source .kt files (platform specific + common) to the unit.
	allSrcs := append(c.inputs.srcs, c.inputs.commonSrcs...)
	if err := c.addRequiredInputs(cu, kzipFile, allSrcs); err != nil {
		return err
	}
	cu.SourceFile = append(cu.SourceFile, allSrcs...)

	// Add the classpath .jars as required files. Skip addition to `SourceFile` since this is external.
	if err := c.addRequiredInputs(cu, kzipFile, c.inputs.classpath); err != nil {
		return err
	}

	// Add the output of this compilation unit
	cu.OutputKey = c.inputs.output

	// Add the kotlin compilation unit in the format expected by the kotlin indexer.
	_, err := kzipFile.AddUnit(cu, nil)
	return err
}

// Registers the inputs as required inputs of the CompilationUnit, and add them to the kzip file.
func (c *compilationUnitGenerator) addRequiredInputs(cu *agp.CompilationUnit, kzipFile *kzip.Writer, inputs []string) error {
	for _, in := range inputs {
		f, err := os.Open(in)
		if err != nil {
			return err
		}
		digest, err := kzipFile.AddFile(f)
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
		mappedVName := c.getMappedVName(in)
		cu.RequiredInput = append(cu.RequiredInput, &agp.CompilationUnit_FileInput{
			Info: &agp.FileInfo{
				Path:   in,
				Digest: digest,
			},
			VName: mappedVName,
		})
	}
	return nil
}

func (c *compilationUnitGenerator) getMappedVName(filePath string) *sgp.VName {
	for _, mapping := range c.vnameMappings {
		re, err := regexp.Compile(mapping.Pattern)
		if err != nil {
			errors.New("Invalid regex in vnames.json")
		}
		matches := re.FindStringSubmatch(filePath)
		if matches != nil {
			vname := mapping.VName

			// Replace placeholders like @1@, @2@, etc., with the corresponding capture groups
			// matches[0] is the full match, so start from index 1
			for i, match := range matches[1:] {
				placeholder := fmt.Sprintf("@%d@", i+1)
				vname.Path = strings.ReplaceAll(vname.Path, placeholder, match)
			}

			// Use the default corpus if none is specified in vname
			if vname.Corpus == "" {
				vname.Corpus = c.inputs.corpus
			}

			return &vname
		}
	}
	return &sgp.VName{
		Path:   filePath,
		Corpus: c.inputs.corpus,
	}
}

// Returns the args passed to kotlinc. This includes the source files and jars on classpath.
func (c *compilationUnitGenerator) getKotlincArgs() []string {
	var ret []string
	// All the additional args
	for _, a := range c.inputs.additionalArgs {
		ret = append(ret, a)
	}
	// classpath
	if len(c.inputs.classpath) > 0 {
		ret = append(ret, "-cp", strings.Join(c.inputs.classpath, ":"))
	}
	// Add the common srcs with "-Xcommon-sources"
	for _, commonSrc := range c.inputs.commonSrcs {
		ret = append(ret, "-Xcommon-sources", commonSrc)
	}
	// add all the source .kt files
	ret = append(ret, c.inputs.srcs...)
	ret = append(ret, c.inputs.commonSrcs...)
	return ret
}
