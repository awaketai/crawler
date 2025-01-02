package main

import (
	"flag"
	"fmt"
)

var (
	BuildTS = "None"
	GitHash = "None"
	GitBranch = "None"
	Version = "None"
)

func GetVersion() string {
	if GitHash != "" {
		h := GitHash
		if len(h) > 7 {
			h = h[:7]
		}

		return fmt.Sprintf("%s-%s",Version,h)
	}

	return Version
}

func Printer(){
	fmt.Println("Version:         ",GetVersion())
	fmt.Println("Git Branch:      ",GitBranch)
	fmt.Println("Git Commit:      ",BuildTS)
	fmt.Println("Build Time (UTC):",GitHash)
}

var PrintVersion = flag.Bool("version",false,"print version")