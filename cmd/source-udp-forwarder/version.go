/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"runtime"
)

var (
	// VERSION, BUILD_DATE, GIT_COMMIT are filled in by the build script
	VERSION     = "<Will be added by go build>"
	COMMIT_SHA1 = "<Will be added by go build>"
	BUILD_DATE  = "<Will be added by go build>"
)

func getVersion() string {
	return fmt.Sprintf("Version: %s, Commit SHA: %s, Build Date: %s, Go Version: %s", VERSION, COMMIT_SHA1, BUILD_DATE, runtime.Version())
}
