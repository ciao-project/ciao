//
// Copyright (c) 2016 Intel Corporation
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

package templateutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

// TemplateFunctionHelp contains formatted documentation that describes the
// additional functions that templateutils adds to Go's templating language.
const TemplateFunctionHelp = `
Some new functions have been added to Go's template language

- tojson outputs the specified object in json format, e.g., {{tojson .}}
- filter operates on an slice or array of structures.  It allows the caller
  to filter the input array based on the value of a single field.
  The function returns a slice containing only the objects that satisfy the
  filter, e.g.

  {{$x := filter . "Protected" "true"}}{{len $x}}

  outputs the number of protected images maintained by the image service.
- filterContains operates along the same lines as filter, but returns
  substring matches

  {{$x := filterContains . "Name" "Cloud"}}{{range $x}}{{.ID}}{{end}}

  outputs the IDs of the workloads which have Cloud in their name.
- filterHasPrefix along the same lines as filter, but returns prefix matches
- filterHasSuffix along the same lines as filter, but returns suffix matches
- filterFolded along the same lines as filter, but  returns matches based
  on equality under Unicode case-folding
- filterRegexp along the same lines as filter, but  returns matches based
  on regular expression matching

  {{$x := filterRegexp . "Name" "^Docker[ a-zA-z]*latest$"}}{{range $x}}{{println .ID .Name}}{{end}}

  outputs the IDs of the workloads which have Docker prefix and latest suffix
  in their name.
- select operates on a slice of structs.  It outputs the value of a specified
  field for each struct on a new line , e.g.,

  {{select . "Name"}}
`

var funcMap = template.FuncMap{
	"filter":          filterByField,
	"filterContains":  filterByContains,
	"filterHasPrefix": filterByHasPrefix,
	"filterHasSuffix": filterByHasSuffix,
	"filterFolded":    filterByFolded,
	"filterRegexp":    filterByRegexp,
	"tojson":          toJSON,
	"select":          selectField,
}

func findField(fieldPath []string, v reflect.Value) reflect.Value {
	f := v
	for _, seg := range fieldPath {
		f = f.FieldByName(seg)
		if f.Kind() == reflect.Ptr {
			f = reflect.Indirect(f)
		}
	}
	return f
}

func filterField(obj interface{}, field, val string, cmp func(string, string) bool) (retval interface{}) {
	defer func() {
		err := recover()
		if err != nil {
			panic(fmt.Errorf("Invalid use of filter: %v", err))
		}
	}()

	list := reflect.ValueOf(obj)
	if list.Kind() == reflect.Ptr {
		list = reflect.Indirect(list)
	}
	filtered := reflect.MakeSlice(list.Type(), 0, list.Len())

	fieldPath := strings.Split(field, ".")

	for i := 0; i < list.Len(); i++ {
		v := list.Index(i)
		if v.Kind() == reflect.Ptr {
			v = reflect.Indirect(v)
		}

		f := findField(fieldPath, v)

		strVal := fmt.Sprintf("%v", f.Interface())
		if cmp(strVal, val) {
			filtered = reflect.Append(filtered, list.Index(i))
		}
	}

	retval = filtered.Interface()
	return

}

func filterByField(obj interface{}, field, val string) (retval interface{}) {
	return filterField(obj, field, val, func(a, b string) bool {
		return a == b
	})
}

func filterByContains(obj interface{}, field, val string) (retval interface{}) {
	return filterField(obj, field, val, strings.Contains)
}

func filterByFolded(obj interface{}, field, val string) (retval interface{}) {
	return filterField(obj, field, val, strings.EqualFold)
}

func filterByHasPrefix(obj interface{}, field, val string) (retval interface{}) {
	return filterField(obj, field, val, strings.HasPrefix)
}

func filterByHasSuffix(obj interface{}, field, val string) (retval interface{}) {
	return filterField(obj, field, val, strings.HasSuffix)
}

func filterByRegexp(obj interface{}, field, val string) (retval interface{}) {
	return filterField(obj, field, val, func(a, b string) bool {
		matched, err := regexp.MatchString(b, a)
		if err != nil {
			panic(fmt.Errorf("Invalid regexp: %v", err))
		}
		return matched
	})
}

func selectField(obj interface{}, field string) string {
	defer func() {
		err := recover()
		if err != nil {
			panic(fmt.Errorf("Invalid use of select: %v", err))
		}
	}()

	var b bytes.Buffer
	list := reflect.ValueOf(obj)
	if list.Kind() == reflect.Ptr {
		list = reflect.Indirect(list)
	}

	fieldPath := strings.Split(field, ".")

	for i := 0; i < list.Len(); i++ {
		v := list.Index(i)
		if v.Kind() == reflect.Ptr {
			v = reflect.Indirect(v)
		}

		f := findField(fieldPath, v)

		fmt.Fprintf(&b, "%v\n", f.Interface())
	}

	return string(b.Bytes())
}

func toJSON(obj interface{}) string {
	b, err := json.MarshalIndent(obj, "", "\t")
	if err != nil {
		return ""
	}
	return string(b)
}

// OutputToTemplate executes the template, whose source is contained within the
// tmplSrc parameter, on the object obj.  The name of the template is given by
// the name parameter.  The results of the execution are printed to os.Stdout.
// All the additional functions provided by templateutils are available to the
// template source code specified in tmplSrc.
func OutputToTemplate(name, tmplSrc string, obj interface{}) error {
	t, err := template.New(name).Funcs(funcMap).Parse(tmplSrc)
	if err != nil {
		return err
	}
	if err = t.Execute(os.Stdout, obj); err != nil {
		return err
	}
	return nil
}

// CreateTemplate creates a new template, whose source is contained within the
// tmplSrc parameter and whose name is given by the name parameter.  All the
// additional functions provided by templateutils are available to the template
// source code specified in tmplSrc.
func CreateTemplate(name, tmplSrc string) (*template.Template, error) {
	if tmplSrc == "" {
		return nil, fmt.Errorf("template %s contains no source", name)
	}

	return template.New(name).Funcs(funcMap).Parse(tmplSrc)
}

func exportedFields(typ reflect.Type) bool {
	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).PkgPath == "" {
			return true
		}
	}

	return false
}

func ignoreKind(kind reflect.Kind) bool {
	return (kind == reflect.Chan) || (kind == reflect.Invalid)
}

func generateStruct(buf io.Writer, typ reflect.Type) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" || ignoreKind(field.Type.Kind()) {
			continue
		}
		fmt.Fprintf(buf, "%s ", field.Name)
		tag := ""
		if field.Tag != "" {
			tag = fmt.Sprintf("`%s`", field.Tag)
		}
		generateUsage(buf, field.Type, tag)
	}
}

func generateUsage(buf io.Writer, typ reflect.Type, tag string) {
	kind := typ.Kind()
	if ignoreKind(kind) {
		return
	}

	switch kind {
	case reflect.Struct:
		if exportedFields(typ) {
			fmt.Fprintf(buf, "struct {\n")
			generateStruct(buf, typ)
			fmt.Fprintf(buf, "}%s\n", tag)
		} else if typ.Name() != "" {
			fmt.Fprintf(buf, "%s%s\n", typ.String(), tag)
		} else {
			fmt.Fprintf(buf, "struct {\n}%s\n", tag)
		}
	case reflect.Slice:
		fmt.Fprintf(buf, "[]")
		generateUsage(buf, typ.Elem(), tag)
	case reflect.Array:
		fmt.Fprintf(buf, "[%d]", typ.Len())
		generateUsage(buf, typ.Elem(), tag)
	case reflect.Map:
		fmt.Fprintf(buf, "map[%s]", typ.Key().String())
		generateUsage(buf, typ.Elem(), tag)
	default:
		fmt.Fprintf(buf, "%s%s\n", typ.String(), tag)
	}
}

func formatType(buf *bytes.Buffer, unformattedType []byte) {
	const typePrefix = "type x "
	source := bytes.NewBufferString(typePrefix)
	_, _ = source.Write(unformattedType)
	formattedType, err := format.Source(source.Bytes())
	if err != nil {
		panic(fmt.Errorf("formatType created invalid Go code: %v", err))
	}
	_, _ = buf.Write(formattedType[len(typePrefix):])
}

func generateIndentedUsage(buf *bytes.Buffer, i interface{}) {
	var source bytes.Buffer
	generateUsage(&source, reflect.TypeOf(i), "")
	formatType(buf, source.Bytes())
}

// GenerateUsageUndecorated returns a formatted string identifying the
// elements of the type of object i that can be accessed  from inside a template.
// Unexported struct values and channels are output are they cannot be usefully
// accessed inside a template.  For example, given
//
//  i := struct {
//      X     int
//      Y     string
//		hidden  float64
//		Invalid chan int
//  }
//
// GenerateUsageUndecorated would return
//
// struct {
//      X     int
//      Y     string
// }
func GenerateUsageUndecorated(i interface{}) string {
	var buf bytes.Buffer
	generateIndentedUsage(&buf, i)
	return buf.String()
}

// GenerateUsageDecorated is similar to GenerateUsageUndecorated with the
// exception that it outputs the TemplateFunctionHelp string after describing
// the type of i.
func GenerateUsageDecorated(flag string, i interface{}) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf,
		"The template passed to the -%s option operates on a\n\n",
		flag)

	generateIndentedUsage(&buf, i)
	fmt.Fprintf(&buf, TemplateFunctionHelp)
	return buf.String()
}
