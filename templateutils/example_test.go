// Copyright (c) 2017 Intel Corporation
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

package templateutils

import (
	"fmt"
	"os"
)

func ExampleGenerateUsageDecorated() {
	cfg := NewConfig(OptCols)
	help := GenerateUsageDecorated("-f", []struct{ X, Y int }{}, cfg)
	fmt.Println(help)
	// output:
	// The template passed to the --f option operates on a
	//
	// []struct {
	//	X int
	//	Y int
	// }
	//
	// Some new functions have been added to Go's template language
	//
	// - 'cols' can be used to extract certain columns from a table consisting of a
	//   slice or array of structs.  It returns a new slice of structs which contain
	//   only the fields requested by the caller.   For example, given a slice of structs
	//
	//   {{cols . "Name" "Address"}}
	//
	//   returns a new slice of structs, each element of which is a structure with only
	//   two fields, 'Name' and 'Address'.
}

func ExampleGenerateUsageUndecorated() {
	i := struct {
		X       int
		Y       string
		hidden  float64
		Invalid chan int
	}{}
	help := GenerateUsageUndecorated(i)
	fmt.Println(help)
	// output:
	// struct {
	// 	X int
	// 	Y string
	// }
}

func ExampleOutputToTemplate() {
	data := []struct{ FirstName, MiddleName, Surname string }{
		{"Marcus", "Tullius", "Cicero"},
		{"Gaius", "Julius", "Caesar"},
		{"Marcus", "Licinius", "Crassus"},
	}

	// print the surname of the person whose middlename is lexographically smallest.
	script := `{{select (head (sort . "MiddleName")) "Surname"}}`
	if err := OutputToTemplate(os.Stdout, "names", script, data, nil); err != nil {
		panic(err)
	}
	// output:
	// Caesar
}
