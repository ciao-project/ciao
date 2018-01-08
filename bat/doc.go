//
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
//

// Package bat provides a set of utility functions that can be used
// to test and manipulate a ciao cluster.  These functions are just wrappers
// around the ciao command.  They invoke ciao commands and parse
// and return the output from these commands in easily consumable go
// values.  Invoking the ciao commands directly, rather than calling
// the REST APIs exposed by ciao's various services, allows us to test
// a little bit more of ciao.
//
// Here's a quick example.  The following code retrieves the instances defined
// on the default tenant and prints out their UUIDs and statuses.
//
//   instances, err := bat.GetAllInstances(context.Background(), "")
//   if err != nil {
//       return err
//   }
//   for uuid, instance := range instances {
//       fmt.Printf("%s : %s\n", uuid, instance.Status)
//   }
//
// The bat.GetAllInstances command calls ciao list instances
//
package bat
