package startf

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qfs"
	"go.starlark.net/starlark"
)

func scriptFile(t *testing.T, path string) qfs.File {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return qfs.NewMemfileBytes(path, data)
}

func TestExecScript(t *testing.T) {
	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/tf.star"))

	stdout := &bytes.Buffer{}
	err := ExecScript(ds, SetOutWriter(stdout))
	if err != nil {
		t.Error(err.Error())
		return
	}
	if ds.Transform == nil {
		t.Error("expected transform")
	}

	output, err := ioutil.ReadAll(stdout)
	if err != nil {
		t.Fatal(err)
	}
	expect := `🤖  running transform...
hello world!`
	if string(output) != expect {
		t.Errorf("stdout mismatch. expected: '%s', got: '%s'", expect, string(output))
	}

	entryReader, err := dsio.NewEntryReader(ds.Structure, ds.BodyFile())
	if err != nil {
		t.Errorf("couldn't create entry reader from returned dataset & body file: %s", err.Error())
		return
	}

	i := 0
	dsio.EachEntry(entryReader, func(_ int, x dsio.Entry, e error) error {
		if e != nil {
			t.Errorf("entry %d iteration error: %s", i, e.Error())
		}
		i++
		return nil
	})

	if i != 8 {
		t.Errorf("expected 8 entries, got: %d", i)
	}
}

func TestExecScript2(t *testing.T) {

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"foo":["bar","baz","bat"]}`))
	}))

	ds := &dataset.Dataset{
		Transform: &dataset.Transform{},
	}
	ds.Transform.SetScriptFile(scriptFile(t, "testdata/fetch.star"))
	err := ExecScript(ds, func(o *ExecOpts) {
		o.Globals["test_server_url"] = starlark.String(s.URL)
	})

	if err != nil {
		t.Error(err.Error())
		return
	}
	if ds.Transform == nil {
		t.Error("expected transform")
	}
}
