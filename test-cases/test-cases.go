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

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"text/tabwriter"
	"text/template"
)

type PackageInfo struct {
	Name   string   `json:"name"`
	Path   string   `json:"path"`
	Files  []string `json:"files"`
	XFiles []string `json:"xfiles"`
}

type TestInfo struct {
	Name           string
	Summary        string
	Description    string
	ExpectedResult string
	Pass           bool
	Result         string
	TimeTaken      string
}

type PackageTests struct {
	Name     string
	Coverage string
	Tests    []*TestInfo
}

type testResults struct {
	result    string
	timeTaken string
}

type colouredRow struct {
	ansiSeq string
	columns []string
}

const goListTemplate = `{
"name" : "{{.ImportPath}}",
"path" : "{{.Dir}}",
"files" : [ {{range $index, $elem := .TestGoFiles }}{{if $index}}, "{{$elem}}"{{else}}"{{$elem}}"{{end}}{{end}} ],
"xfiles" : [ {{range $index, $elem := .XTestGoFiles }}{{if $index}}, "{{$elem}}"{{else}}"{{$elem}}"{{end}}{{end}} ]
},
`

const htmlTemplate = `
<html>
<head>
<title>Test Cases</title>
<style type="text/css">
{{.CSS}}
</style>
</head>
<body>
{{range .Tests}}
<h1>{{.Name}}</h1>
<p><i>Coverage: {{.Coverage}}</i></p>
<table style="table-layout:fixed" border="1">
<tr><th style="width:10%">Name</th><th style="width:20%">Summary</th><th style="width:30%">Description</th><th style="width:20%">ExpectedResult</th><th style="width:10%">Result</th><th style="width:10%">Time Taken</th></tr>
{{range .Tests}}
<tr {{if .Pass}}style="color: green"{{else}}style="color: red"{{end}}><td>{{.Name}}</td><td>{{.Summary}}</td><td>{{.Description}}</td><td>{{.ExpectedResult}}</td><td>{{.Result}}</td><td>{{.TimeTaken}}</td></tr>
{{end}}
</table>
{{end}}
</body>
</html>
`

var resultRegexp *regexp.Regexp
var coverageRegexp *regexp.Regexp

var cssPath string
var textOutput bool
var short bool
var tags string
var colour bool

func init() {
	flag.StringVar(&cssPath, "css", "", "Full path to CSS file")
	flag.BoolVar(&textOutput, "text", false, "Output text instead of HTML")
	flag.BoolVar(&short, "short", false, "If true -short is passed to go test")
	flag.StringVar(&tags, "tags", "", "Build tags to pass to go test")
	flag.BoolVar(&colour, "colour", true, "If true failed tests are coloured red in text mode")
	resultRegexp = regexp.MustCompile(`--- (FAIL|PASS): ([^\s]+) \(([^\)]+)\)`)
	coverageRegexp = regexp.MustCompile(`^coverage: ([^\s]+)`)
}

func parseCommentGroup(ti *TestInfo, comment string) {
	groups := regexp.MustCompile("\n\n").Split(comment, 4)
	fields := []*string{&ti.Summary, &ti.Description, &ti.ExpectedResult}
	for i, c := range groups {
		*fields[i] = c
	}
}

func isTestingFunc(decl *ast.FuncDecl) bool {
	if !strings.HasPrefix(decl.Name.String(), "Test") {
		return false
	}

	paramList := decl.Type.Params.List
	if len(paramList) != 1 {
		return false
	}

	recType, ok := paramList[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}

	pt, ok := recType.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	id, ok := pt.X.(*ast.Ident)
	if !ok {
		return false
	}

	return id.Name == "testing" && pt.Sel.Name == "T"
}

func parseTestFile(filePath string) ([]*TestInfo, error) {
	tests := make([]*TestInfo, 0, 32)
	fs := token.NewFileSet()
	tr, err := parser.ParseFile(fs, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, decl := range tr.Decls {
		if decl, ok := decl.(*ast.FuncDecl); ok {
			if !isTestingFunc(decl) {
				continue
			}

			ti := &TestInfo{Name: decl.Name.String()}
			tests = append(tests, ti)

			if decl.Doc == nil {
				continue
			}

			parseCommentGroup(ti, decl.Doc.Text())
		}
	}

	return tests, nil
}

func extractTests(packages []PackageInfo) []*PackageTests {
	pts := make([]*PackageTests, 0, len(packages))
	for _, p := range packages {
		if len(p.Files) == 0 || strings.Contains(p.Name, "/vendor/") {
			continue
		}
		packageTest := &PackageTests{
			Name: p.Name,
		}

		files := make([]string, 0, len(p.Files)+len(p.XFiles))
		files = append(files, p.Files...)
		files = append(files, p.XFiles...)
		for _, f := range files {
			filePath := path.Join(p.Path, f)
			ti, err := parseTestFile(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse %s: %s\n",
					filePath, err)
				continue
			}
			packageTest.Tests = append(packageTest.Tests, ti...)
		}
		pts = append(pts, packageTest)
	}
	return pts
}

func findTestFiles(packs []string) ([]PackageInfo, error) {
	var output bytes.Buffer
	fmt.Fprintln(&output, "[")
	listArgs := []string{"list", "-f", goListTemplate}
	listArgs = append(listArgs, packs...)
	cmd := exec.Command("go", listArgs...)
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	lastComma := bytes.LastIndex(output.Bytes(), []byte{','})
	if lastComma != -1 {
		output.Truncate(lastComma)
	}
	fmt.Fprintln(&output, "]")
	var testPackages []PackageInfo
	err = json.Unmarshal(output.Bytes(), &testPackages)
	if err != nil {
		return nil, err
	}
	return testPackages, nil
}

func runPackageTests(p *PackageTests) int {
	var output bytes.Buffer
	var coverage string

	exitCode := 0
	results := make(map[string]*testResults)
	args := []string{"test", p.Name, "-v", "-cover"}
	if short {
		args = append(args, "-short")
	}
	if tags != "" {
		args = append(args, "-tags", tags)
	}

	cmd := exec.Command("go", args...)
	cmd.Stdout = &output
	_ = cmd.Run()

	scanner := bufio.NewScanner(&output)
	for scanner.Scan() {
		line := scanner.Text()
		matches := resultRegexp.FindStringSubmatch(line)
		if matches != nil && len(matches) == 4 {
			results[matches[2]] = &testResults{matches[1], matches[3]}
			continue
		}

		if coverage == "" {
			matches := coverageRegexp.FindStringSubmatch(line)
			if matches == nil || len(matches) != 2 {
				continue
			}
			coverage = matches[1]
		}
	}

	for _, t := range p.Tests {
		res := results[t.Name]
		if res == nil {
			t.Result = "NOT RUN"
			t.TimeTaken = "N/A"
			exitCode = 1
		} else {
			t.Result = res.result
			t.Pass = res.result == "PASS"
			if !t.Pass {
				exitCode = 1
			}
			t.TimeTaken = res.timeTaken
		}
	}

	if coverage != "" {
		p.Coverage = coverage
	} else {
		p.Coverage = "Unknown"
	}

	return exitCode
}

func identifyPackages(packs []string) []string {
	if len(packs) == 0 {
		packs = []string{"."}
	} else if len(packs) > 1 {
		for _, p := range packs {
			if p == "./..." {
				packs = []string{p}
				break
			}
		}
	}
	return packs
}

func generateHTMLReport(tests []*PackageTests) error {
	var css string
	if cssPath != "" {
		cssBytes, err := ioutil.ReadFile(cssPath)
		if err != nil {
			log.Printf("Unable to read css file %s : %v",
				cssPath, err)
		} else {
			css = string(cssBytes)
		}
	}

	tmpl, err := template.New("tests").Parse(htmlTemplate)
	if err != nil {
		log.Fatalf("Unable to parse html template: %s\n", err)
	}

	return tmpl.Execute(os.Stdout, &struct {
		Tests []*PackageTests
		CSS   string
	}{
		tests,
		css,
	})
}

func findCommonPrefix(tests []*PackageTests) string {
	if len(tests) == 0 {
		return ""
	}

	pkgName := tests[0].Name
OUTER:
	for {
		index := strings.LastIndex(pkgName, "/")
		if index == -1 {
			return ""
		}
		pkgName := pkgName[:index+1]

		var i int
		for i = 1; i < len(tests); i++ {
			if !strings.HasPrefix(tests[i].Name, pkgName) {
				continue OUTER
			}
		}
		return pkgName
	}
}

func generateColourTextReport(tests []*PackageTests) {
	prefix := findCommonPrefix(tests)
	table := make([]colouredRow, 0, 128)
	table = append(table, colouredRow{
		"",
		[]string{"Package", "Test Case", "Time Taken", "Result"},
	})
	colWidth := []int{0, 0, 0, 0}
	for i := range colWidth {
		colWidth[i] = len(table[0].columns[i])
	}

	coloured := false
	for _, p := range tests {
		pkgName := p.Name[len(prefix):]
		for _, t := range p.Tests {
			row := colouredRow{}
			if !t.Pass {
				row.ansiSeq = fmt.Sprintf("%c[%dm", 0x1b, 31)
				coloured = true
			} else if t.Pass && coloured {
				coloured = false
				row.ansiSeq = fmt.Sprintf("%c[%dm", 0x1b, 0)
			}
			row.columns = []string{pkgName, t.Name, t.TimeTaken, t.Result}
			for i := range colWidth {
				if colWidth[i] < len(row.columns[i]) {
					colWidth[i] = len(row.columns[i])
				}
			}
			table = append(table, row)
		}
	}

	for _, row := range table {
		fmt.Printf("%s", row.ansiSeq)
		for i, col := range row.columns {
			fmt.Printf(col)
			fmt.Printf("%s", strings.Repeat(" ", colWidth[i]-len(col)))
			fmt.Printf(" ")
		}
		fmt.Println("")
	}

	if coloured {
		fmt.Printf("%c[%dm\n", 0x1b, 0)
	}
}

func generateTextReport(tests []*PackageTests) {
	prefix := findCommonPrefix(tests)
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, ' ', 0)
	fmt.Fprintln(w, "Package\tTest Case\tTime Taken\tResult\t")
	for _, p := range tests {
		pkgName := p.Name[len(prefix):]
		for _, t := range p.Tests {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", pkgName,
				t.Name, t.TimeTaken, t.Result)
		}
	}
	_ = w.Flush()
	fmt.Println()
}

func main() {

	flag.Parse()

	packs := identifyPackages(flag.Args())

	packages, err := findTestFiles(packs)
	if err != nil {
		log.Fatalf("Unable to discover test files: %s", err)
	}

	tests := extractTests(packages)
	exitCode := 0
	for _, p := range tests {
		exitCode = exitCode | runPackageTests(p)
	}

	if textOutput {
		if colour {
			generateColourTextReport(tests)
		} else {
			generateTextReport(tests)
		}
	} else {
		err = generateHTMLReport(tests)
	}

	if err != nil {
		log.Fatalf("Unable to generate report: %s\n", err)
	}

	os.Exit(exitCode)
}
