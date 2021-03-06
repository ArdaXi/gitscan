package checks

import "github.com/ardaxi/gitscan/providers"

type Result struct {
	File        providers.File
	Caption     string
	Description string
}

type Check func(providers.File) []*Result

var Checks []Check
