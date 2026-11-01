// Command antvet checks ant code for mistakes the compiler cannot see. Run it
// through go vet:
//
//	go install github.com/skudra-go/ant/tools/antvet/cmd/antvet@latest
//	go vet -vettool=$(which antvet) ./...
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"github.com/skudra-go/ant/tools/antvet/analyzer"
)

func main() { unitchecker.Main(analyzer.Analyzer) }
