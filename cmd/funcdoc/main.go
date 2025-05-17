package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/ghetzel/diecast"
	"github.com/ghetzel/go-stockutil/rxutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var rxFnDocString = `//\s*fn\s*(?P<func>[^:]+):\s*(?P<docstring>.*)`

type docArg struct {
	Arg  string
	Type reflect.Type
}

type functionDoc struct {
	Name      string
	DocString string
	Signature string
	Returns   string
}

type functionDocSet []*functionDoc

func (self functionDocSet) Len() int {
	return len(self)
}
func (self functionDocSet) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}
func (self functionDocSet) Less(i, j int) bool {
	return self[i].Name < self[j].Name
}

func GenerateFunctionDocs(funcs diecast.FuncMap, sourcefile string) (functionDocSet, error) {
	var docs = make(functionDocSet, 0)

	if source, err := parser.ParseFile(token.NewFileSet(), sourcefile, nil, parser.ParseComments); err == nil {
	NextComment:
		for _, group := range source.Comments {
			var doc = &functionDoc{}

			for _, comment := range group.List {
				if match := rxutil.Match(rxFnDocString, comment.Text); match != nil {
					var fnname = match.Group(`func`)
					var docstring = match.Group(`docstring`)

					// if the comment refers to a function we know about, and the docstring
					// portion is not empty, start building the functionDoc struct
					if fn, ok := funcs[fnname]; ok && docstring != `` {
						var argNames []string

						if an := rxutil.Match(`(?:\*(\w+)\*)+`, docstring); an != nil {
							argNames = an.AllCaptures()
						}

						if signature, outputs, err := getFnSignature(fn, argNames); err == nil {
							doc.Name = fnname
							doc.DocString = docstring
							doc.Signature = signature
							doc.Returns = outputs
						} else {
							fmt.Printf("signature failed: %v\n", err)
							continue NextComment
						}
					}
				} else if doc.Name != `` {
					// if doc.Name is set, then we're nigh on a multi-line docstring, so append it.
					doc.DocString += ` ` + strings.TrimSpace(
						strings.TrimPrefix(comment.Text, `//`),
					)
				} else {
					continue NextComment
				}
			}

			docs = append(docs, doc)
		}

		return docs, nil
	} else {
		return nil, err
	}
}

func getFnSignature(fn any, inArgNames []string) (string, string, error) {
	fn = typeutil.ResolveValue(fn)

	if typeutil.IsKind(fn, reflect.Func) {
		var args []string
		var outs []string

		var fnT = reflect.TypeOf(fn)

		// figure out input arguments
		for in := 0; in < fnT.NumIn(); in++ {
			var inT = fnT.In(in)
			var typename = inT.Name()

			switch typename {
			case `any`, ``:
				typename = `any`
			}

			if fnT.IsVariadic() && (in+1) == fnT.NumIn() {
				typename = `[` + typename + ` ..]`
			}

			if in < len(inArgNames) {
				args = append(args, fmt.Sprintf("%s %s", inArgNames[in], typename))
			} else {
				args = append(args, typename)
			}
		}

		// figure out output arguments
		for o := 0; o < fnT.NumOut(); o++ {
			var outT = fnT.Out(o)
			var typename = outT.Name()

			switch typename {
			case `any`:
				typename = `any`
			}

			outs = append(outs, typename)
		}

		var inArgs = strings.Join(args, ` `)
		var outArgs = strings.Join(outs, ` `)

		return inArgs, outArgs, nil
	} else {
		return ``, ``, fmt.Errorf("must provide a function to get a signature")
	}
}

func main() {
	if f, err := os.Open(`docs/functions_pre.md`); err == nil {
		defer f.Close()
		if data, err := io.ReadAll(f); err == nil {
			fmt.Printf("%s\n", string(data))
		}
	}

	var docs functionDocSet
	var standardFuncs = diecast.GetStandardFunctions(nil)

	if funcFiles, err := filepath.Glob(`functions_defs_*.go`); err == nil {
		for _, funcdefs := range funcFiles {
			if d, err := GenerateFunctionDocs(standardFuncs, funcdefs); err == nil {
				docs = append(docs, d...)
			} else {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}

	if d, err := GenerateFunctionDocs(new(diecast.Server).GetTemplateFunctions(nil, nil), `server.go`); err == nil {
		docs = append(docs, d...)
	} else {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}

	sort.Sort(docs)

	fmt.Printf("## Function List\n\n")

	for _, doc := range docs {
		fmt.Printf("- [%s](#%s)\n", doc.Name, doc.Name)
	}

	fmt.Printf("## Function Usage\n\n")

	for _, doc := range docs {
		var returnSignature = doc.Returns

		if returnSignature != `` {
			var outSig = strings.Split(returnSignature, `,`)

			if outSig[len(outSig)-1] == `error` {
				outSig = outSig[:len(outSig)-1]
			}

			returnSignature = strings.Join(outSig, ` `)

			if len(outSig) > 1 {
				returnSignature = `(` + returnSignature + `)`
			}
		}

		fmt.Printf("---\n\n")
		fmt.Printf("<a name=\"%s\"></a>\n", doc.Name)
		fmt.Printf("```\n")
		fmt.Printf("%s %s -> %s\n", doc.Name, doc.Signature, returnSignature)
		fmt.Printf("```\n")

		fmt.Printf("%s\n\n", doc.DocString)
	}

	if f, err := os.Open(`docs/functions_post.md`); err == nil {
		defer f.Close()
		if data, err := io.ReadAll(f); err == nil {
			fmt.Printf("%s\n", string(data))
		}
	}
}
