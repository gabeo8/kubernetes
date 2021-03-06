/*
Copyright 2014 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd"
	"github.com/cpuguy83/go-md2man/mangen"
	"github.com/russross/blackfriday"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func main() {
	// use os.Args instead of "flags" because "flags" will mess up the man pages!
	docsDir := "docs/man/man1/"
	if len(os.Args) == 2 {
		docsDir = os.Args[1]
	} else if len(os.Args) > 2 {
		fmt.Fprintf(os.Stderr, "usage: %s [output directory]\n", os.Args[0])
		os.Exit(1)
	}

	docsDir, err := filepath.Abs(docsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	stat, err := os.Stat(docsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "output directory %s does not exist\n", docsDir)
		os.Exit(1)
	}

	if !stat.IsDir() {
		fmt.Fprintf(os.Stderr, "output directory %s is not a directory\n", docsDir)
		os.Exit(1)
	}
	docsDir = docsDir + "/"

	// Set environment variables used by kubectl so the output is consistent,
	// regardless of where we run.
	os.Setenv("HOME", "/home/username")
	kubectl := cmd.NewFactory(nil).NewKubectlCommand(ioutil.Discard)
	genMarkdown(kubectl, "", docsDir)
	for _, c := range kubectl.Commands() {
		genMarkdown(c, "kubectl", docsDir)
	}
}

func preamble(out *bytes.Buffer, name, short, long string) {
	out.WriteString(`% KUBERNETES(1) kubernetes User Manuals
% Eric Paris
% Jan 2015
# NAME
`)
	fmt.Fprintf(out, "%s \\- %s\n\n", name, short)
	fmt.Fprintf(out, "# SYNOPSIS\n")
	fmt.Fprintf(out, "**%s** [OPTIONS]\n\n", name)
	fmt.Fprintf(out, "# DESCRIPTION\n")
	fmt.Fprintf(out, "%s\n\n", long)
}

func printFlags(out *bytes.Buffer, flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		format := "**--%s**=%s\n\t%s\n\n"
		if flag.Value.Type() == "string" {
			// put quotes on the value
			format = "**--%s**=%q\n\t%s\n\n"
		}
		if len(flag.Shorthand) > 0 {
			format = "**-%s**, " + format
		} else {
			format = "%s" + format
		}
		fmt.Fprintf(out, format, flag.Shorthand, flag.Name, flag.DefValue, flag.Usage)
	})
}

func printOptions(out *bytes.Buffer, command *cobra.Command) {
	var flags *pflag.FlagSet
	if command.HasFlags() {
		flags = command.Flags()
	} else if !command.HasParent() && command.HasPersistentFlags() {
		flags = command.PersistentFlags()
	}
	if flags == nil {
		return
	}
	fmt.Fprintf(out, "# OPTIONS\n")
	printFlags(out, flags)
	fmt.Fprintf(out, "\n")
}

func genMarkdown(command *cobra.Command, parent, docsDir string) {
	dparent := strings.Replace(parent, " ", "-", -1)
	name := command.Name()
	dname := name
	if len(parent) > 0 {
		dname = dparent + "-" + name
		name = parent + " " + name
	}

	out := new(bytes.Buffer)
	short := command.Short
	long := command.Long
	if len(long) == 0 {
		long = short
	}

	preamble(out, name, short, long)
	printOptions(out, command)

	if len(command.Commands()) > 0 || len(parent) > 0 {
		fmt.Fprintf(out, "# SEE ALSO\n")
		if len(parent) > 0 {
			fmt.Fprintf(out, "**%s(1)**, ", dparent)
		}
		for _, c := range command.Commands() {
			fmt.Fprintf(out, "**%s-%s(1)**, ", dname, c.Name())
			genMarkdown(c, name, docsDir)
		}
		fmt.Fprintf(out, "\n")
	}

	out.WriteString(`
# HISTORY
January 2015, Originally compiled by Eric Paris (eparis at redhat dot com) based on the kubernetes source material, but hopefully they have been automatically generated since!
`)

	renderer := mangen.ManRenderer(0)
	extensions := 0
	extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_AUTOLINK
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS
	extensions |= blackfriday.EXTENSION_FOOTNOTES
	extensions |= blackfriday.EXTENSION_TITLEBLOCK

	final := blackfriday.Markdown(out.Bytes(), renderer, extensions)

	filename := docsDir + dname + ".1"
	outFile, err := os.Create(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer outFile.Close()
	_, err = outFile.Write(final)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
