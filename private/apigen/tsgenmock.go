// Copyright (C) 2023 Storj Labs, Inc.
// See LICENSE for copying information.

package apigen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/zeebo/errs"
)

// MustWriteTSMock writes generated TypeScript code into a file indicated by path.
// The generated code is an API client mock to run in the browser.
//
// If an error occurs, it panics.
func (a *API) MustWriteTSMock(path string) {
	rootDir := a.outputRootDir()
	fullpath := filepath.Join(rootDir, path)
	err := os.MkdirAll(filepath.Dir(fullpath), 0700)
	if err != nil {
		panic(errs.Wrap(err))
	}

	f := newTSGenMockFile(fullpath, a)

	f.generateTS()

	err = f.write()
	if err != nil {
		panic(errs.Wrap(err))
	}
}

type tsGenMockFile struct {
	*tsGenFile
}

func newTSGenMockFile(filepath string, api *API) *tsGenMockFile {
	return &tsGenMockFile{
		tsGenFile: newTSGenFile(filepath, api),
	}
}

func (f *tsGenMockFile) generateTS() {
	f.pf("// AUTOGENERATED BY private/apigen")
	f.pf("// DO NOT EDIT.")

	f.registerTypes()
	f.result += f.types.GenerateTypescriptDefinitions()

	f.result += `
class APIError extends Error {
	constructor(
		public readonly msg: string,
		public readonly responseStatusCode?: number,
	) {
		super(msg);
	}
}
`

	for _, group := range f.api.EndpointGroups {
		f.createAPIClient(group)
	}
}

func (f *tsGenMockFile) createAPIClient(group *EndpointGroup) {
	f.pf("\nexport class %sHttpApi%s {", capitalize(group.Name), strings.ToUpper(f.api.Version))
	// Properties.
	f.pf("\tpublic readonly respStatusCode: number;")
	f.pf("")

	// Constructor
	f.pf("\t// When respStatuscode is passed, the client throws an APIError on each method call")
	f.pf("\t// with respStatusCode as HTTP status code.")
	f.pf("\t// respStatuscode must be equal or greater than 400")
	f.pf("\tconstructor(respStatusCode?: number) {")
	f.pf("\t\tif (typeof respStatusCode === 'undefined') {")
	f.pf("\t\t\tthis.respStatusCode = 0;")
	f.pf("\t\t\treturn;")
	f.pf("\t\t}")
	f.pf("")
	f.pf("\t\tif (respStatusCode < 400) {")
	f.pf("\t\t\tthrow new Error('invalid response status code for API Error, it must be greater or equal than 400');")
	f.pf("\t\t}")
	f.pf("")
	f.pf("\t\tthis.respStatusCode = respStatusCode;")
	f.pf("\t}")

	// Methods to call API endpoints.
	for _, method := range group.endpoints {
		f.pf("")

		funcArgs, _ := f.getArgsAndPath(method, group)

		returnType := "void"
		if method.Response != nil {
			if method.ResponseMock == nil {
				panic(
					fmt.Sprintf(
						"ResponseMock is nil and Response isn't nil. Endpoint.Method=%q, Endpoint.Path=%q",
						method.Method, method.Path,
					))
			}

			returnType = TypescriptTypeName(reflect.TypeOf(method.Response))
		}

		f.pf("\tpublic async %s(%s): Promise<%s> {", method.TypeScriptName, funcArgs, returnType)
		f.pf("\t\tif (this.respStatusCode !== 0) {")
		f.pf("\t\t\tthrow new APIError('mock error message: ' + this.respStatusCode, this.respStatusCode);")
		f.pf("\t\t}")
		f.pf("")

		if method.ResponseMock != nil {
			res, err := json.Marshal(method.ResponseMock)
			if err != nil {
				panic(
					fmt.Sprintf(
						"error when marshaling ResponseMock: %+v. Endpoint.Method=%q, Endpoint.Path=%q",
						err, method.Method, method.Path,
					))
			}

			f.pf("\t\treturn JSON.parse('%s') as %s;", string(res), returnType)
		} else {
			f.pf("\t\treturn;")
		}

		f.pf("\t}")
	}
	f.pf("}")
}
