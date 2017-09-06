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
	{helpFilter, helpFilterIndex},
	{helpFilterContains, helpFilterContainsIndex},
	{helpFilterHasPrefix, helpFilterHasPrefixIndex},
	{helpFilterHasSuffix, helpFilterHasSuffixIndex},
	{helpFilterFolded, helpFilterFoldedIndex},
	{helpFilterRegexp, helpFilterRegexpIndex},
	{helpToJSON, helpToJSONIndex},
	{helpSelect, helpSelectIndex},
	{helpSelectAlt, helpSelectAltIndex},
	{helpTable, helpTableIndex},
	{helpTableAlt, helpTableAltIndex},
	{helpTableX, helpTableXIndex},
	{helpTableXAlt, helpTableXAltIndex},
	{helpCols, helpColsIndex},
	{helpSort, helpSortIndex},
	{helpRows, helpRowsIndex},
	{helpHead, helpHeadIndex},
	{helpTail, helpTailIndex},
	{helpDescribe, helpDescribeIndex},
	{helpPromote, helpPromoteIndex},
	{helpSliceof, helpSliceofIndex},
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
