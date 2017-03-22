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
	"reflect"
	"regexp"
	"strings"
	"text/tabwriter"
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
- table outputs a table given an array or a slice of structs.  The table headings are taken
  from the names of the structs fields.  Hidden fields and fields of type channel are ignored.
  The tabwidth and minimum column width are hardcoded to 8.  An example of table's usage is

  {{table .}}
- tablex is similar to table but it allows the caller more control over the table's appearance.
  Users can control the names of the headings and also set the tab and column width.  tablex
  takes 3 or more parameters.  The first parameter is the slice of structs to output, the
  second is the minimum column width, the third the tab width.  The fourth and subsequent
  parameters are the names of the column headings.  The column headings are optional and the
  field names of the structure will be used if they are absent.  Example of its usage are

  {{tablex . 12 8 "Column 1" "Column 2"}}
  {{tablex . 8 8}}
- cols can be used to extract certain columns from a table consisting of a slice or array of
  structs.  It returns a new slice of structs which contain only the fields requested by the
  caller.   For example, given a slice of structs

  {{cols . "Name" "Address"}}

  returns a new slice of structs, each element of which is a structure with only two fields,
  Name and Address.
`

// BUG(markdryan): Tests for all functions
// BUG(markdryan): Check table and cols commands work with pointers to slices and slices of pointers

type tableHeading struct {
	name  string
	index int
}

var funcMap = template.FuncMap{
	"filter":          filterByField,
	"filterContains":  filterByContains,
	"filterHasPrefix": filterByHasPrefix,
	"filterHasSuffix": filterByHasSuffix,
	"filterFolded":    filterByFolded,
	"filterRegexp":    filterByRegexp,
	"tojson":          toJSON,
	"select":          selectField,
	"table":           table,
	"tablex":          tablex,
	"cols":            cols,
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

func assertCollectionOfStructs(obj interface{}) {
	typ := reflect.TypeOf(obj)
	kind := typ.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		panic("slice or an array of structs expected")
	}
	styp := typ.Elem()
	if styp.Kind() != reflect.Struct {
		panic("slice or an array of structs expected")
	}
}

func getTableHeadings(obj interface{}) []tableHeading {
	assertCollectionOfStructs(obj)

	typ := reflect.TypeOf(obj)
	styp := typ.Elem()

	var headings []tableHeading
	for i := 0; i < styp.NumField(); i++ {
		field := styp.Field(i)
		if field.PkgPath != "" || ignoreKind(field.Type.Kind()) {
			continue
		}
		headings = append(headings, tableHeading{name: field.Name, index: i})
	}

	if len(headings) == 0 {
		panic("structures must contain at least one exported non-channel field")
	}
	return headings
}

func createTable(obj interface{}, minWidth, tabWidth int, headings []tableHeading) string {
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, minWidth, tabWidth, 1, ' ', 0)
	for _, h := range headings {
		fmt.Fprintf(w, "%s\t", h.name)
	}
	fmt.Fprintln(w)

	v := reflect.ValueOf(obj)
	for i := 0; i < v.Len(); i++ {
		el := v.Index(i)
		for _, h := range headings {
			fmt.Fprintf(w, "%v\t", el.Field(h.index).Interface())
		}
		fmt.Fprintln(w)
	}
	_ = w.Flush()

	return b.String()
}

func table(obj interface{}) string {
	return createTable(obj, 8, 8, getTableHeadings(obj))
}

func tablex(obj interface{}, minWidth, tabWidth int, userHeadings ...string) string {
	headings := getTableHeadings(obj)
	if len(headings) < len(userHeadings) {
		panic(fmt.Sprintf("Too many headings specified.  Max permitted %d got %d",
			len(headings), len(userHeadings)))
	}
	for i := range userHeadings {
		headings[i].name = userHeadings[i]
	}
	return createTable(obj, minWidth, tabWidth, headings)
}

func cols(obj interface{}, fields ...string) interface{} {
	assertCollectionOfStructs(obj)
	if len(fields) == 0 {
		panic("at least one column name must be specified")
	}

	var newFields []reflect.StructField
	var indicies []int
	styp := reflect.TypeOf(obj).Elem()
	for i := 0; i < styp.NumField(); i++ {
		field := styp.Field(i)
		if field.PkgPath != "" || ignoreKind(field.Type.Kind()) {
			continue
		}

		var j int
		for j = 0; j < len(fields); j++ {
			if fields[j] == field.Name {
				break
			}
		}
		if j == len(fields) {
			continue
		}

		indicies = append(indicies, i)
		newFields = append(newFields, field)
	}

	if len(indicies) != len(fields) {
		panic("not all column names are valid")
	}

	val := reflect.ValueOf(obj)
	newStyp := reflect.StructOf(newFields)
	newVal := reflect.MakeSlice(reflect.SliceOf(newStyp), val.Len(), val.Len())
	for i := 0; i < val.Len(); i++ {
		sval := val.Index(i)
		newSval := reflect.New(newStyp).Elem()
		for j, origIndex := range indicies {
			newSval.Field(j).Set(sval.Field(origIndex))
		}
		newVal.Index(i).Set(newSval)
	}

	return newVal.Interface()
}

// OutputToTemplate executes the template, whose source is contained within the
// tmplSrc parameter, on the object obj.  The name of the template is given by
// the name parameter.  The results of the execution are output to w.
// All the additional functions provided by templateutils are available to the
// template source code specified in tmplSrc.
func OutputToTemplate(w io.Writer, name, tmplSrc string, obj interface{}) error {
	t, err := template.New(name).Funcs(funcMap).Parse(tmplSrc)
	if err != nil {
		return err
	}
	if err = t.Execute(w, obj); err != nil {
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
