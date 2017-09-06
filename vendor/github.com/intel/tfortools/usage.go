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

package tfortools

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"reflect"
)

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
	case reflect.Ptr:
		fmt.Fprintf(buf, "*")
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

func dumpMethods(buf *bytes.Buffer, typ reflect.Type) {
	var i int

	for i = 0; i < typ.NumMethod(); i++ {
		if typ.Method(i).PkgPath == "" {
			break
		}
	}

	if i == typ.NumMethod() {
		return
	}

	fmt.Fprintf(buf, "\nMethods:\n\n")

	for i = 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		if m.PkgPath != "" {
			continue
		}
		typ := m.Type

		fmt.Fprintf(buf, "%s(", m.Name)
		if typ.NumIn() > 1 {
			fmt.Fprintf(buf, "%v", typ.In(1))
			for j := 2; j < typ.NumIn(); j++ {
				fmt.Fprintf(buf, ", %v", typ.In(j))
			}
		}
		fmt.Fprintf(buf, ")")
		if typ.NumOut() == 1 {
			fmt.Fprintf(buf, " %v", typ.Out(0))
		} else if typ.NumOut() > 1 {
			fmt.Fprintf(buf, " (")
			fmt.Fprintf(buf, typ.Out(0).String())
			for j := 1; j < typ.NumOut(); j++ {
				fmt.Fprintf(buf, ", %v", typ.Out(j))
			}
			fmt.Fprintf(buf, ")")
		}
		fmt.Fprintln(buf)
	}
}

func generateIndentedUsage(buf *bytes.Buffer, i interface{}) {
	var source bytes.Buffer
	typ := reflect.TypeOf(i)

	generateUsage(&source, typ, "")
	formatType(buf, source.Bytes())

	if typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ)
	}
	dumpMethods(buf, typ)
}
