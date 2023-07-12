package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/ashmrtn/allowtags/pkg/allowtags"
)

func main() {
	a := allowtags.New()
	singlechecker.Main(a)
}
