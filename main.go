package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type replacer struct {
	r_package       *regexp.Regexp
	t_go_pkg        *template.Template
	require_package bool
}

func (r *replacer) scan(dir string) error {
	fileInfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, info := range fileInfos {
		if info.IsDir() {
			if err := r.scan(filepath.Join(dir, info.Name())); err != nil {
				return err
			}
		} else if strings.HasSuffix(info.Name(), ".proto") {
			if err := r.replace(filepath.Join(dir, info.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

var regexPackageDirective = regexp.MustCompile(`package\s+(.*)\s*;`)
var regexOption = regexp.MustCompile(`([ \t]*)\boption\s+([^\s]+)\s*=\s*"((:?[^"\\]|\\.)*)"\s*;`)

func (r *replacer) replace(filename string) error {
	fmt.Printf("  file: %v\n", filename)
	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	packageDirectiveMatches := regexPackageDirective.FindSubmatchIndex(buf)
	if len(packageDirectiveMatches) == 0 {
		if r.require_package {
			return fmt.Errorf("no package definition found: %v", filename)
		} else {
			fmt.Printf("    no package definition")
			return nil
		}
	}

	packageDirective := string(buf[packageDirectiveMatches[2]:packageDirectiveMatches[3]])
	packageNameMatches := r.r_package.FindStringSubmatch(packageDirective)
	if len(packageNameMatches) == 0 {
		fmt.Printf("    package didn't match: %v\n", packageNameMatches[1])
		return nil
	}
	for i, _ := range packageNameMatches {
		packageNameMatches[i] = strings.ReplaceAll(packageNameMatches[i], ".", "/")
	}

	outBuf := bytes.Buffer{}
	if err := r.t_go_pkg.Execute(&outBuf, packageNameMatches); err != nil {
		return err
	}
	fmt.Printf("    result: %v\n", outBuf.String())

	optionMatches := regexOption.FindAllSubmatchIndex(buf, -1)
	var goPackageMatch []int
	for _, indices := range optionMatches {
		opt_name := string(buf[indices[4]:indices[5]])
		if opt_name == "go_package" {
			goPackageMatch = indices
		}
	}

	result := bytes.Buffer{}
	if goPackageMatch == nil {
		// There was no previous definition for option "go_package". We have to insert.
		// Find a suitable insertion point using the following strategy:
		//  - Iterate matched options, insert before the first option that would succeed "go_package" in
		//    lexical order.
		//  - If this didn't result in an inserted option, insert after the last option.
		//  - If there was no last option, insert after package directive.

		inserted := false
		for _, indices := range optionMatches {
			opt_name := string(buf[indices[4]:indices[5]])
			if strings.Compare("go_package", opt_name) > 0 {
				// Too early to insert
				continue
			}
			// Insert before this option
			_, _ = result.Write(buf[:indices[0]])
			_, _ = result.Write(buf[indices[2]:indices[3]]) // repeat indent
			_, _ = fmt.Fprintf(&result, "option go_package = \"%v\";\n", outBuf.String())
			_, _ = result.Write(buf[indices[0]:])
			inserted = true
			break
		}

		if !inserted && optionMatches != nil {
			// Insert after last option
			lastOption := optionMatches[len(optionMatches) - 1]
			_, _ = result.Write(buf[:lastOption[1]])
			_, _ = result.Write([]byte("\n"))
			_, _ = result.Write(buf[lastOption[2]:lastOption[3]]) // repeat indent
			_, _ = fmt.Fprintf(&result, "option go_package = \"%v\";", outBuf.String())
			_, _ = result.Write(buf[lastOption[1]:])
			inserted = true
		}

		if !inserted {
			// Insert after package directive
			_, _ = result.Write(buf[:packageDirectiveMatches[1]])
			_, _ = fmt.Fprintf(&result, "\n\noption go_package = \"%v\";", outBuf.String())
			_, _ = result.Write(buf[packageDirectiveMatches[1]:])
		}
	} else {
		// replace existing option go_package declaration
		_, _ = result.Write(buf[0:goPackageMatch[6]])
		_, _ = outBuf.WriteTo(&result)
		_, _ = result.Write(buf[goPackageMatch[7]:])
	}

	f, err = os.Create(filename)
	if err != nil {
		return err
	}

	_, err = result.WriteTo(f)
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	pkg := flag.String("package", "(.*)", "Regex for matching the package")
	go_package := flag.String("go_package", "github.com/example/example/{{index . 1}}",
		"Pattern of go_package to be set")
	flag.Parse()

	r := replacer{
		r_package: regexp.MustCompile(*pkg),
		t_go_pkg:  template.Must(template.New("go_package").Parse(*go_package)),
	}

	for _, path := range flag.Args() {
		fmt.Printf("Replacing go_package recursively in in %v\n", path)
		if err := r.scan(path); err != nil {
			fmt.Printf("Error: %v", err)
		}
	}

}
