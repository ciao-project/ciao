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
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"
)

// BUG(markdryan): Tests for all functions
// BUG(markdryan): Need Go doc
// BUG(markdryan): Map to slice
// BUG(markdryan): 3rd party extensions
// BUG(markdryan): Split into smaller files

// These constants are used to ensure that all the help text
// for functions provided by this package are always presented
// in the same order.
const (
	helpFilterIndex = iota
	helpFilterContainsIndex
	helpFilterHasPrefixIndex
	helpFilterHasSuffixIndex
	helpFilterFoldedIndex
	helpFilterRegexpIndex
	helpToJSONIndex
	helpSelectIndex
	helpTableIndex
	helpTableXIndex
	helpColsIndex
	helpSortIndex
	helpRowsIndex
	helpHeadIndex
	helpTailIndex
)

// UsageError indicates that one of the functions provided by this package
// has been used incorrectly by a Go template program.
type UsageError string

func (e UsageError) Error() string {
	return string(e)
}

type funcHelpInfo struct {
	help  string
	index int
}

// Config is used to specify which functions should be added Go's template
// language.  It's not necessary to create a Config option.  Nil can be passed
// to all templateutils functions that take a Context object indicating the
// default behaviour is desired.  However, if you wish to restrict the
// number of functions added to Go's template language or you want to add your
// own functions, you'll need to create a Config object.  This can be done
// using the NewConfig function.
//
// All members of Config are private.
type Config struct {
	funcMap  template.FuncMap
	funcHelp []funcHelpInfo
}

func (c *Config) Len() int           { return len(c.funcHelp) }
func (c *Config) Swap(i, j int)      { c.funcHelp[i], c.funcHelp[j] = c.funcHelp[j], c.funcHelp[i] }
func (c *Config) Less(i, j int) bool { return c.funcHelp[i].index < c.funcHelp[j].index }

const helpFilter = `- 'filter' operates on an slice or array of structures.  It allows the caller
  to filter the input array based on the value of a single field.
  The function returns a slice containing only the objects that satisfy the
  filter, e.g.

  {{len (filter . "Protected" "true")}}

  outputs the number of elements whose "Protected" field is equal to "true".
`

// OptFilter indicates that the filter function should be enabled.
// 'filter' operates on an slice or array of structures.  It allows the caller
// to filter the input array based on the value of a single field.
// The function returns a slice containing only the objects that satisfy the
// filter, e.g.
//
// {{len (filter . "Protected" "true")}}
//
// outputs the number of elements whose "Protected" field is equal to "true".
func OptFilter(c *Config) {
	c.funcMap["filter"] = filterByField
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpFilter, helpFilterIndex})
}

const helpFilterContains = `- 'filterContains' operates along the same lines as filter, but returns
  substring matches

  {{len(filterContains . "Name" "Cloud"}})

  outputs the number of elements whose "Name" field contains the word "Cloud".
`

// OptFilterContains indicates that the filterContains function should be
// enabled.  'filterContains' operates along the same lines as filter, but returns
// substring matches
//
// {{len(filterContains . "Name" "Cloud"}})
//
// outputs the number of elements whose "Name" field contains the word "Cloud".
func OptFilterContains(c *Config) {
	c.funcMap["filterContains"] = filterByContains
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpFilterContains, helpFilterContainsIndex})
}

const helpFilterHasPrefix = `- 'filterHasPrefix' is similar to filter, but returns prefix matches.
`

// OptFilterHasPrefix indicates that the filterHasPrefix function should be enabled.
// 'filterHasPrefix' is similar to filter, but returns prefix matches.
func OptFilterHasPrefix(c *Config) {
	c.funcMap["filterHasPrefix"] = filterByHasPrefix
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpFilterHasPrefix, helpFilterHasPrefixIndex})
}

const helpFilterHasSuffix = `- 'filterHasSuffix' is similar to filter, but returns suffix matches.
`

// OptFilterHasSuffix indicates that the filterHasSuffix function should be enabled.
// 'filterHasSuffix' is similar to filter, but returns suffix matches.
func OptFilterHasSuffix(c *Config) {
	c.funcMap["filterHasSuffix"] = filterByHasSuffix
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpFilterHasSuffix, helpFilterHasSuffixIndex})
}

const helpFilterFolded = `- 'filterFolded' is similar to filter, but returns matches based on equality
  under Unicode case-folding.
`

// OptFilterFolded indicates that the filterFolded function should be enabled.
// 'filterFolded' is similar to filter, but returns matches based on equality
// under Unicode case-folding.
func OptFilterFolded(c *Config) {
	c.funcMap["filterFolded"] = filterByFolded
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpFilterFolded, helpFilterFoldedIndex})
}

const helpFilterRegexp = `- 'filterRegexp' is similar to filter, but returns matches based on regular
  expression matching

  {{len (filterRegexp . "Name" "^Docker[ a-zA-z]*latest$"}})

  outputs the number of elements whose "Name" field have 'Docker' as a prefix
  and 'latest' as a suffix in their name.
`

// OptFilterRegexp indicates that the filterRegexp function should be enabled.
// 'filterRegexp' is similar to filter, but returns matches based on regular
// expression matching
//
//  {{len (filterRegexp . "Name" "^Docker[ a-zA-z]*latest$"}})
//
// outputs the number of elements whose "Name" field have 'Docker' as a prefix
// and 'latest' as a suffix in their name.
func OptFilterRegexp(c *Config) {
	c.funcMap["filterRegexp"] = filterByRegexp
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpFilterRegexp, helpFilterRegexpIndex})
}

// OptAllFilters is a convenience function that enables the following functions;
// 'filter', 'filterContains', 'filterHasPrefix', 'filterHasSuffix', 'filterFolded',
// and 'filterRegexp'
func OptAllFilters(c *Config) {
	OptFilter(c)
	OptFilterContains(c)
	OptFilterHasPrefix(c)
	OptFilterHasSuffix(c)
	OptFilterFolded(c)
	OptFilterRegexp(c)
}

const helpToJSON = `- 'tojson' outputs the target object in json format, e.g., {{tojson .}}
`

// OptToJSON indicates that the 'tosjon' function should be enabled.
// 'tojson' outputs the target object in json format, e.g., {{tojson .}}
func OptToJSON(c *Config) {
	c.funcMap["tojson"] = toJSON
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpToJSON, helpToJSONIndex})
}

const helpSelect = `- 'select' operates on a slice of structs.  It outputs the value of a specified
  field for each struct on a new line , e.g.,

  {{select . "Name"}}

  prints the 'Name' field of each structure in the slice.
`

// OptSelect indicates that the 'select' function should be enabled.
// 'select' operates on a slice of structs.  It outputs the value of a specified
// field for each struct on a new line , e.g.,
//
// {{select . "Name"}}
//
// prints the 'Name' field of each structure in the slice.
func OptSelect(c *Config) {
	c.funcMap["select"] = selectField
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpSelect, helpSelectIndex})
}

const helpTable = `- 'table' outputs a table given an array or a slice of structs.  The table
  headings are taken from the names of the structs fields.  Hidden fields and
  fields of type channel are ignored.  The tabwidth and minimum column width
  are hardcoded to 8.  An example of table's usage is

  {{table .}}
`

// OptTable indicates that the 'table' function should be enabled.
// 'table' outputs a table given an array or a slice of structs.  The table
// headings are taken from the names of the structs fields.  Hidden fields and
// fields of type channel are ignored.  The tabwidth and minimum column width
// are hardcoded to 8.  An example of table's usage is
//
//  {{table .}}
func OptTable(c *Config) {
	c.funcMap["table"] = table
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpTable, helpTableIndex})
}

const helpTableX = `- 'tablex' is similar to table but it allows the caller more control over the
  table's appearance.  Users can control the names of the headings and also set
  the tab and column width.  'tablex' takes 3 or more parameters.  The first
  parameter is the slice of structs to output, the second is the minimum column
  width, the third the tab width.  The fourth and subsequent parameters are the
  names of the column headings.  The column headings are optional and the field
  names of the structure will be used if they are absent.  Example of its usage
  are:

  {{tablex . 12 8 "Column 1" "Column 2"}}
  {{tablex . 8 8}}
`

// OptTableX indicates that the 'tablex' function should be enabled.
// 'tablex' is similar to table but it allows the caller more control over the
// table's appearance.  Users can control the names of the headings and also set
// the tab and column width.  'tablex' takes 3 or more parameters.  The first
// parameter is the slice of structs to output, the second is the minimum column
// width, the third the tab width.  The fourth and subsequent parameters are the
// names of the column headings.  The column headings are optional and the field
// names of the structure will be used if they are absent.  Example of its usage
// are:
//
//  {{tablex . 12 8 "Column 1" "Column 2"}}
//  {{tablex . 8 8}}
func OptTableX(c *Config) {
	c.funcMap["tablex"] = tablex
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpTableX, helpTableXIndex})
}

const helpCols = `- 'cols' can be used to extract certain columns from a table consisting of a
  slice or array of structs.  It returns a new slice of structs which contain
  only the fields requested by the caller.   For example, given a slice of structs

  {{cols . "Name" "Address"}}

  returns a new slice of structs, each element of which is a structure with only
  two fields, 'Name' and 'Address'.
`

// OptCols indicates that the 'cols' function should be enabled.
// 'cols' can be used to extract certain columns from a table consisting of a
// slice or array of structs.  It returns a new slice of structs which contain
// only the fields requested by the caller.   For example, given a slice of structs
//
//  {{cols . "Name" "Address"}}
//
// returns a new slice of structs, each element of which is a structure with only
// two fields, 'Name' and 'Address'.
func OptCols(c *Config) {
	c.funcMap["cols"] = cols
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpCols, helpColsIndex})
}

const helpSort = `- 'sort' sorts a slice or an array of structs.  It takes three parameters.  The
  first is the slice; the second is the name of the structure field by which to
  'sort'; the third provides the direction of the 'sort'.  The third parameter is
  optional.  If provided, it must be either "asc" or "dsc".  If omitted the
  elements of the slice are sorted in ascending order.  The type of the second
  field can be a number or a string.  When presented with another type, 'sort'
  will try to sort the elements by the string representation of the chosen field.
  The following example sorts a slice in ascending order by the Name field.
  
  {{sort . "Name"}}
`

// OptSort indicates that the 'sort' function should be enabled.
// 'sort' sorts a slice or an array of structs.  It takes three parameters.  The
// first is the slice; the second is the name of the structure field by which to
// 'sort'; the third provides the direction of the 'sort'.  The third parameter is
// optional.  If provided, it must be either "asc" or "dsc".  If omitted the
// elements of the slice are sorted in ascending order.  The type of the second
// field can be a number or a string.  When presented with another type, 'sort'
// will try to sort the elements by the string representation of the chosen field.
// The following example sorts a slice in ascending order by the Name field.
//
//  {{sort . "Name"}}
func OptSort(c *Config) {
	c.funcMap["sort"] = sortSlice
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpSort, helpSortIndex})
}

const helpRows = `- 'rows' is used to extract a set of given rows from a slice or an array.  It
  takes at least two parameters. The first is the slice on which to operate.
  All subsequent parameters must be integers that correspond to a row in the
  input slice.  Indicies that refer to non-existent rows are ignored.  For
  example:

  {{rows . 1 2}}

  extracts the 2nd and 3rd rows from the slice represented by '.'.
`

// OptRows indicates that the 'rows' function should be enabled.
// 'rows' is used to extract a set of given rows from a slice or an array.  It
// takes at least two parameters. The first is the slice on which to operate.
// All subsequent parameters must be integers that correspond to a row in the
// input slice.  Indicies that refer to non-existent rows are ignored.  For
// example:
//
//  {{rows . 1 2}}
//
//  extracts the 2nd and 3rd rows from the slice represented by '.'.
func OptRows(c *Config) {
	c.funcMap["rows"] = rows
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpRows, helpRowsIndex})
}

const helpHead = `- 'head' operates on a slice or an array, returning the first n elements of
  that array as a new slice.  If n is not provided, a slice containing the
  first element of the input slice is returned.  For example,

  {{ head .}}

  returns a single element slice containing the first element of '.' and

  {{ head . 3}}

  returns a slice containing the first three elements of '.'.  If '.' contains
  only 2 elements the slice returned by {{ head . 3}} would be identical to the
  input slice.
`

// OptHead indicates that the 'head' function should be enabled.
// 'head' operates on a slice or an array, returning the first n elements of
// that array as a new slice.  If n is not provided, a slice containing the
// first element of the input slice is returned.  For example,
//
//  {{ head .}}
//
// returns a single element slice containing the first element of '.' and
//
//  {{ head . 3}}
//
// returns a slice containing the first three elements of '.'.  If '.' contains
// only 2 elements the slice returned by {{ head . 3}} would be identical to the
// input slice.
func OptHead(c *Config) {
	c.funcMap["head"] = head
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpHead, helpHeadIndex})
}

const helpTail = `- 'tail' is similar to head except that it returns a slice containing the last
  n elements of the input slice.  For example,

  {{tail . 2}}

  returns a new slice containing the last two elements of '.'.
`

// OptTail indicates that the 'tail' function should be enabled.
// 'tail' is similar to head except that it returns a slice containing the last
// n elements of the input slice.  For example,
//
//  {{tail . 2}}
//
// returns a new slice containing the last two elements of '.'.
func OptTail(c *Config) {
	c.funcMap["tail"] = tail
	c.funcHelp = append(c.funcHelp, funcHelpInfo{helpTail, helpTailIndex})
}

// NewConfig creates a new Config object that can be passed to other functions
// in this package.  The Config option keeps track of which new functions are
// added to Go's template libray.  If this function is call without arguments,
// all the functions defined in this package are enabled in the resulting Config
// object.  To control which functions get added specify some options, e.g.,
//
// ctx := templateutils.NewConfig(templateutils.OptHead, templateutils.OptTail)
//
// creates a new Config object that enables the 'head' and 'tail' functions only.
func NewConfig(options ...func(*Config)) *Config {
	if len(options) == 0 {
		return &Config{
			funcMap:  funcMap,
			funcHelp: funcHelpSlice,
		}
	}

	c := &Config{
		funcMap: make(template.FuncMap),
	}
	for _, f := range options {
		f(c)
	}
	sort.Sort(c)

	return c
}

// TemplateFunctionHelp generates formatted documentation that describes the
// additional functions that the Config object c adds to Go's templating language.
// If c is nil, documentation is generated for all functions provided by
// templateutils.
func TemplateFunctionHelp(c *Config) string {
	var helpers []funcHelpInfo
	if c == nil {
		helpers = funcHelpSlice
	} else {
		helpers = c.funcHelp
	}

	b := &bytes.Buffer{}
	_, _ = b.WriteString("Some new functions have been added to Go's template language\n\n")

	for _, h := range helpers {
		_, _ = b.WriteString(h.help)
	}
	return b.String()
}

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
	"sort":            sortSlice,
	"rows":            rows,
	"head":            head,
	"tail":            tail,
}

var funcHelpSlice = []funcHelpInfo{
	{helpFilter, helpFilterIndex},
	{helpFilterContains, helpFilterContainsIndex},
	{helpFilterHasPrefix, helpFilterHasPrefixIndex},
	{helpFilterHasSuffix, helpFilterHasSuffixIndex},
	{helpFilterFolded, helpFilterFoldedIndex},
	{helpFilterRegexp, helpFilterRegexpIndex},
	{helpToJSON, helpToJSONIndex},
	{helpSelect, helpSelectIndex},
	{helpTable, helpTableIndex},
	{helpTableX, helpTableXIndex},
	{helpCols, helpColsIndex},
	{helpSort, helpSortIndex},
	{helpRows, helpRowsIndex},
	{helpHead, helpHeadIndex},
	{helpTail, helpTailIndex},
}

var sortAscMap = map[reflect.Kind]func(interface{}, interface{}) bool{
	reflect.Int: func(v1, v2 interface{}) bool {
		return v1.(int) < v2.(int)
	},
	reflect.Int8: func(v1, v2 interface{}) bool {
		return v1.(int8) < v2.(int8)
	},
	reflect.Int16: func(v1, v2 interface{}) bool {
		return v1.(int16) < v2.(int16)
	},
	reflect.Int32: func(v1, v2 interface{}) bool {
		return v1.(int32) < v2.(int32)
	},
	reflect.Int64: func(v1, v2 interface{}) bool {
		return v1.(int64) < v2.(int64)
	},
	reflect.Uint: func(v1, v2 interface{}) bool {
		return v1.(uint) < v2.(uint)
	},
	reflect.Uint8: func(v1, v2 interface{}) bool {
		return v1.(uint8) < v2.(uint8)
	},
	reflect.Uint16: func(v1, v2 interface{}) bool {
		return v1.(uint16) < v2.(uint16)
	},
	reflect.Uint32: func(v1, v2 interface{}) bool {
		return v1.(uint32) < v2.(uint32)
	},
	reflect.Uint64: func(v1, v2 interface{}) bool {
		return v1.(uint64) < v2.(uint64)
	},
	reflect.Float64: func(v1, v2 interface{}) bool {
		return v1.(float64) < v2.(float64)
	},
	reflect.Float32: func(v1, v2 interface{}) bool {
		return v1.(float32) < v2.(float32)
	},
	reflect.String: func(v1, v2 interface{}) bool {
		return v1.(string) < v2.(string)
	},
}

var sortDscMap = map[reflect.Kind]func(interface{}, interface{}) bool{
	reflect.Int: func(v1, v2 interface{}) bool {
		return v2.(int) < v1.(int)
	},
	reflect.Int8: func(v1, v2 interface{}) bool {
		return v2.(int8) < v1.(int8)
	},
	reflect.Int16: func(v1, v2 interface{}) bool {
		return v2.(int16) < v1.(int16)
	},
	reflect.Int32: func(v1, v2 interface{}) bool {
		return v2.(int32) < v1.(int32)
	},
	reflect.Int64: func(v1, v2 interface{}) bool {
		return v2.(int64) < v1.(int64)
	},
	reflect.Uint: func(v1, v2 interface{}) bool {
		return v2.(uint) < v1.(uint)
	},
	reflect.Uint8: func(v1, v2 interface{}) bool {
		return v2.(uint8) < v1.(uint8)
	},
	reflect.Uint16: func(v1, v2 interface{}) bool {
		return v2.(uint16) < v1.(uint16)
	},
	reflect.Uint32: func(v1, v2 interface{}) bool {
		return v2.(uint32) < v1.(uint32)
	},
	reflect.Uint64: func(v1, v2 interface{}) bool {
		return v2.(uint64) < v1.(uint64)
	},
	reflect.Float64: func(v1, v2 interface{}) bool {
		return v2.(float64) < v1.(float64)
	},
	reflect.Float32: func(v1, v2 interface{}) bool {
		return v2.(float32) < v1.(float32)
	},
	reflect.String: func(v1, v2 interface{}) bool {
		return v2.(string) < v1.(string)
	},
}

type valueSorter struct {
	val   reflect.Value
	field int
	less  func(v1, v2 interface{}) bool
}

func (v *valueSorter) Len() int {
	return v.val.Len()
}

func (v *valueSorter) Less(i, j int) bool {
	iVal := v.val.Index(i)
	jVal := v.val.Index(j)
	return v.less(iVal.Field(v.field).Interface(), jVal.Field(v.field).Interface())
}

func (v *valueSorter) Swap(i, j int) {
	iVal := v.val.Index(i).Interface()
	jVal := v.val.Index(j).Interface()
	v.val.Index(i).Set(reflect.ValueOf(jVal))
	v.val.Index(j).Set(reflect.ValueOf(iVal))
}

func getValue(obj interface{}) reflect.Value {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	return val
}

func newValueSorter(obj interface{}, field string, ascending bool) *valueSorter {
	val := reflect.ValueOf(obj)
	typ := reflect.TypeOf(obj)
	sTyp := typ.Elem()

	var index int
	var fTyp reflect.StructField
	for index = 0; index < sTyp.NumField(); index++ {
		fTyp = sTyp.Field(index)
		if fTyp.Name == field {
			break
		}
	}
	if index == sTyp.NumField() {
		panic(UsageError(fmt.Sprintf("%s is not a valid field name", field)))
	}
	fKind := fTyp.Type.Kind()

	var lessFn func(interface{}, interface{}) bool
	if ascending {
		lessFn = sortAscMap[fKind]
	} else {
		lessFn = sortDscMap[fKind]
	}
	if lessFn == nil {
		var stringer *fmt.Stringer
		if !fTyp.Type.Implements(reflect.TypeOf(stringer).Elem()) {
			panic(UsageError(fmt.Sprintf("cannot sort fields of type %s", fKind)))
		}
		lessFn = func(v1, v2 interface{}) bool {
			return v1.(fmt.Stringer).String() < v2.(fmt.Stringer).String()
		}
	}
	return &valueSorter{
		val:   val,
		field: index,
		less:  lessFn,
	}
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
			panic(UsageError(fmt.Sprintf("Invalid use of filter: %v", err)))
		}
	}()

	list := getValue(obj)
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
			panic(UsageError(fmt.Sprintf("Invalid regexp: %v", err)))
		}
		return matched
	})
}

func selectField(obj interface{}, field string) string {
	defer func() {
		err := recover()
		if err != nil {
			panic(UsageError(fmt.Sprintf("Invalid use of select: %v", err)))
		}
	}()

	var b bytes.Buffer
	list := getValue(obj)

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

func assertCollectionOfStructs(v reflect.Value) {
	typ := v.Type()
	kind := typ.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		panic("slice or an array of structs expected")
	}
	styp := typ.Elem()
	if styp.Kind() != reflect.Struct {
		panic("slice or an array of structs expected")
	}
}

func getTableHeadings(v reflect.Value) []tableHeading {
	assertCollectionOfStructs(v)

	typ := v.Type()
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
		panic(UsageError("structures must contain at least one exported non-channel field"))
	}
	return headings
}

func createTable(v reflect.Value, minWidth, tabWidth int, headings []tableHeading) string {
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, minWidth, tabWidth, 1, ' ', 0)
	for _, h := range headings {
		fmt.Fprintf(w, "%s\t", h.name)
	}
	fmt.Fprintln(w)

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
	val := getValue(obj)
	return createTable(val, 8, 8, getTableHeadings(val))
}

func tablex(obj interface{}, minWidth, tabWidth int, userHeadings ...string) string {
	val := getValue(obj)
	headings := getTableHeadings(val)
	if len(headings) < len(userHeadings) {
		panic(UsageError(fmt.Sprintf("Too many headings specified.  Max permitted %d got %d",
			len(headings), len(userHeadings))))
	}
	for i := range userHeadings {
		headings[i].name = userHeadings[i]
	}
	return createTable(val, minWidth, tabWidth, headings)
}

func cols(obj interface{}, fields ...string) interface{} {
	val := getValue(obj)
	assertCollectionOfStructs(val)
	if len(fields) == 0 {
		panic(UsageError("at least one column name must be specified"))
	}

	var newFields []reflect.StructField
	var indicies []int
	styp := val.Type().Elem()
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
		panic(UsageError("not all column names are valid"))
	}

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

func sortSlice(obj interface{}, field string, direction ...string) interface{} {
	ascending := true
	if len(direction) > 1 {
		panic(UsageError("Too many parameters passed to sort"))
	} else if len(direction) == 1 {
		if direction[0] == "dsc" {
			ascending = false
		} else if direction[0] != "asc" {
			panic(UsageError("direction parameter must be \"asc\" or \"dsc\""))
		}
	}

	val := getValue(obj)
	assertCollectionOfStructs(val)

	copy := reflect.MakeSlice(reflect.SliceOf(val.Type().Elem()), 0, val.Len())
	for i := 0; i < val.Len(); i++ {
		copy = reflect.Append(copy, val.Index(i))
	}

	newobj := copy.Interface()
	vs := newValueSorter(newobj, field, ascending)
	sort.Sort(vs)
	return newobj
}

func rows(obj interface{}, rows ...int) interface{} {
	val := getValue(obj)
	typ := val.Type()
	kind := typ.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		panic(UsageError("slice or an array of expected"))
	}

	if len(rows) == 0 {
		panic(UsageError("at least one row index must be specified"))
	}

	copy := reflect.MakeSlice(reflect.SliceOf(val.Type().Elem()), 0, len(rows))
	for _, row := range rows {
		if row < val.Len() {
			copy = reflect.Append(copy, val.Index(row))
		}
	}

	return copy.Interface()
}

func assertSliceAndRetrieveCount(obj interface{}, count ...int) (reflect.Value, int) {
	val := getValue(obj)
	typ := val.Type()
	kind := typ.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		panic(UsageError("slice or an array of expected"))
	}

	rows := 1
	if len(count) == 1 {
		rows = count[0]
	} else if len(count) > 1 {
		panic(UsageError("accepts a maximum of two arguments expected"))
	}

	return val, rows
}

func head(obj interface{}, count ...int) interface{} {
	val, rows := assertSliceAndRetrieveCount(obj, count...)
	copy := reflect.MakeSlice(reflect.SliceOf(val.Type().Elem()), 0, rows)
	for i := 0; i < rows && i < val.Len(); i++ {
		copy = reflect.Append(copy, val.Index(i))
	}

	return copy.Interface()
}

func tail(obj interface{}, count ...int) interface{} {
	val, rows := assertSliceAndRetrieveCount(obj, count...)
	copy := reflect.MakeSlice(reflect.SliceOf(val.Type().Elem()), 0, rows)
	start := val.Len() - rows
	if start < 0 {
		start = 0
	}
	for i := start; i < val.Len(); i++ {
		copy = reflect.Append(copy, val.Index(i))
	}

	return copy.Interface()
}

func getFuncMap(cfg *Config) template.FuncMap {
	var fmap template.FuncMap
	if cfg == nil {
		fmap = funcMap
	} else {
		fmap = cfg.funcMap
	}
	return fmap
}

// OutputToTemplate executes the template, whose source is contained within the
// tmplSrc parameter, on the object obj.  The name of the template is given by
// the name parameter.  The results of the execution are output to w.
// The functions enabled in the cfg parameter will be made available to the
// template source code specified in tmplSrc.  If cfg is nil, all the
// additional functions provided by templateutils will be enabled.  Any errors
// caused by the incorrect usage of one of the functions provided by this package
// will result in a templateutils.UsageError.
func OutputToTemplate(w io.Writer, name, tmplSrc string, obj interface{}, cfg *Config) (err error) {
	defer func() {
		if err1 := recover(); err1 != nil {
			if err1, ok := err1.(UsageError); ok {
				err = err1
			} else {
				panic(err1)
			}
		}
	}()
	t, err := template.New(name).Funcs(getFuncMap(cfg)).Parse(tmplSrc)
	if err != nil {
		return err
	}
	if err = t.Execute(w, obj); err != nil {
		return err
	}
	return nil
}

// CreateTemplate creates a new template, whose source is contained within the
// tmplSrc parameter and whose name is given by the name parameter. The functions
// enabled in the cfg parameter will be made available to the template source code
// specified in tmplSrc.  If cfg is nil, all the additional functions provided by
// templateutils will be enabled.  Any errors caused by the incorrect usage of
// one of the functions provided by this package will result a panic when the
// when the template is executed.  The type of the object associated with the
// panic is of type templateutils.UsageError.
func CreateTemplate(name, tmplSrc string, cfg *Config) (*template.Template, error) {
	if tmplSrc == "" {
		return nil, fmt.Errorf("template %s contains no source", name)
	}

	return template.New(name).Funcs(getFuncMap(cfg)).Parse(tmplSrc)
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
// exception that it outputs the usage information for all the new functions
// enabled in the Config object cfg.  If cfg is nil, help information is
// printed for all new template functions defined by this package.
func GenerateUsageDecorated(flag string, i interface{}, cfg *Config) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf,
		"The template passed to the -%s option operates on a\n\n",
		flag)

	generateIndentedUsage(&buf, i)
	fmt.Fprintln(&buf)
	fmt.Fprintf(&buf, TemplateFunctionHelp(cfg))
	return buf.String()
}
