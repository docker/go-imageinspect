// Copyright 2022 go-imageinspect authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package imageinspect

type Provenance struct { // TODO: this is only a stub, to be refactored later
	BuildSource     string            `json:",omitempty"`
	BuildDefinition string            `json:",omitempty"`
	BuildParameters map[string]string `json:",omitempty"`
	Materials       []Material
}

type Material struct {
	Type  string `json:",omitempty"`
	Ref   string `json:",omitempty"`
	Alias string `json:",omitempty"`
	Pin   string `json:",omitempty"`
}
