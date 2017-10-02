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

import "text/template"

var funcMap = template.FuncMap{
	"filter":          filterByField,
	"filterContains":  filterByContains,
	"filterHasPrefix": filterByHasPrefix,
	"filterHasSuffix": filterByHasSuffix,
	"filterFolded":    filterByFolded,
	"filterRegexp":    filterByRegexp,
	"tojson":          toJSON,
	"select":          selectField,
	"selectalt":       selectFieldAlt,
	"table":           table,
	"tablealt":        tableAlt,
	"tablex":          tablex,
	"tablexalt":       tablexAlt,
	"cols":            cols,
	"sort":            sortSlice,
	"rows":            rows,
	"head":            head,
	"tail":            tail,
	"describe":        describe,
	"promote":         promote,
	"sliceof":         sliceof,
}

var funcHelpSlice = []funcHelpInfo{
	{"filter", helpFilter, helpFilterIndex},
	{"filterContains", helpFilterContains, helpFilterContainsIndex},
	{"filterHasPrefix", helpFilterHasPrefix, helpFilterHasPrefixIndex},
	{"filterHasSuffix", helpFilterHasSuffix, helpFilterHasSuffixIndex},
	{"filterFolded", helpFilterFolded, helpFilterFoldedIndex},
	{"filterRegexp", helpFilterRegexp, helpFilterRegexpIndex},
	{"tojson", helpToJSON, helpToJSONIndex},
	{"select", helpSelect, helpSelectIndex},
	{"selectalt", helpSelectAlt, helpSelectAltIndex},
	{"table", helpTable, helpTableIndex},
	{"tablealt", helpTableAlt, helpTableAltIndex},
	{"tablex", helpTableX, helpTableXIndex},
	{"tablexalt", helpTableXAlt, helpTableXAltIndex},
	{"cols", helpCols, helpColsIndex},
	{"sort", helpSort, helpSortIndex},
	{"rows", helpRows, helpRowsIndex},
	{"head", helpHead, helpHeadIndex},
	{"tail", helpTail, helpTailIndex},
	{"describe", helpDescribe, helpDescribeIndex},
	{"promote", helpPromote, helpPromoteIndex},
	{"sliceof", helpSliceof, helpSliceofIndex},
}

func getFuncMap(cfg *Config) template.FuncMap {
	if cfg == nil {
		return funcMap
	}

	return cfg.funcMap
}

func getHelpers(cfg *Config) []funcHelpInfo {
	if cfg == nil {
		return funcHelpSlice
	}

	return cfg.funcHelp
}
