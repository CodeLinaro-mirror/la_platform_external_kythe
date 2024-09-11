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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"unicode"
)

func maybeLogErrAndExit(err error, message string) {
	if err != nil {
		fmt.Printf("%s: %v\n", message, err)
		os.Exit(1)
	}
}

// Assumes `ins` is a list of paths.
// If any of the element starts with `@`, it returns the list of paths contained in that file.
func maybeParseRsp(ins []string) ([]string, error) {
	var ret []string
	for _, in := range ins {
		if strings.HasPrefix(in, "@") {
			// rsp file
			f, err := os.Open(strings.TrimPrefix(in, "@"))
			if err != nil {
				return nil, err
			}
			files, err := readRspFile(f)
			if err != nil {
				return nil, err
			}
			ret = append(ret, files...)
		} else {
			ret = append(ret, in)
		}
	}
	return ret, nil
}

func readRspFile(r io.Reader) ([]string, error) {
	var files []string
	var file []byte

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	for _, c := range buf {
		switch {
		case unicode.IsSpace(rune(c)):
			// Separator
			if len(file) != 0 {
				files = append(files, string(file))
			}
			// Reset
			file = []byte{}
		default:
			// Add the character to the `file`
			file = append(file, c)
		}
	}

	if len(file) != 0 {
		files = append(files, string(file))
	}

	return files, nil
}
