// Package skytf implements dataset transformations using the skylark programming dialect
// For more info on skylark check github.com/google/skylark
package skytf

import (
	"fmt"
	"log"

	"github.com/google/skylark"
	"github.com/google/skylark/repl"
	"github.com/google/skylark/resolve"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
)

// ExecOpts defines options for exection
type ExecOpts struct {
	AllowFloat     bool // allow floating-point numbers
	AllowSet       bool // allow set data type
	AllowLambda    bool // allow lambda expressions
	AllowNestedDef bool // allow nested def statements
}

// DefaultExecOpts applies default options to an ExecOpts pointer
func DefaultExecOpts(o *ExecOpts) {
	o.AllowFloat = true
	o.AllowSet = true
}

// ExecFile executes a transformation against a filepath
func ExecFile(ds *dataset.Dataset, filename string, opts ...func(o *ExecOpts)) (dsio.EntryReader, error) {
	o := &ExecOpts{}
	DefaultExecOpts(o)
	for _, opt := range opts {
		opt(o)
	}

	resolve.AllowFloat = o.AllowFloat
	resolve.AllowSet = o.AllowSet
	resolve.AllowLambda = o.AllowLambda
	resolve.AllowNestedDef = o.AllowNestedDef

	if ds.Transform == nil {
		ds.Transform = &dataset.Transform{}
	}
	ds.Transform.Syntax = "skylark"

	cm := commit{}
	cf := newConfig(ds)
	hr := newHTTPRequests(ds)
	skylark.Universe["commit"] = skylark.NewBuiltin("commit", cm.Do)
	skylark.Universe["get_config"] = skylark.NewBuiltin("get_config", cf.GetConfig)
	skylark.Universe["fetch_json_url"] = skylark.NewBuiltin("fetch_json_url", hr.FetchJSONUrl)

	thread := &skylark.Thread{Load: repl.MakeLoad()}
	// globals := make(skylark.StringDict)

	// Execute specified file.
	var err error
	_, err = skylark.ExecFile(thread, filename, nil, nil)
	if err != nil {
		log.Print(err.Error())
		return nil, err
	}

	if !cm.called {
		return nil, fmt.Errorf("commit must be called once to add data")
	}

	// Print the global environment.
	// var names []string
	// for name := range globals {
	// 	if !strings.HasPrefix(name, "_") {
	// 		names = append(names, name)
	// 	}
	// }
	// sort.Strings(names)
	// for _, name := range names {
	// 	fmt.Fprintf(os.Stderr, "%s = %s\n", name, globals[name])
	// }
	sch := dataset.BaseSchemaArray
	if cm.data.Type() == "dict" {
		sch = dataset.BaseSchemaObject
	}

	st := &dataset.Structure{
		Format: dataset.UnknownDataFormat,
		Schema: sch,
	}

	if ds.Structure == nil {
		ds.Structure = st
	}

	// fmt.Printf("%v", cm.data)

	r := NewEntryReader(st, cm.data)
	return r, nil
}
