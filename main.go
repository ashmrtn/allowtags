package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/ashmrtn/keytags/pkg/analyzer"
)

func main() {
	a := analyzer.NewKeyTags()
	singlechecker.Main(a)
}
