package qri

import (
	"fmt"

	"github.com/google/skylark"
	"github.com/google/skylark/skylarkstruct"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/skytf/lib"
)

// ModuleName defines the expected name for this module when used
// in skylark's load() function, eg: load('qri.sky', 'qri')
const ModuleName = "qri.sky"

// NewModule creates a new qri module instance
func NewModule(ds *dataset.Dataset, secrets map[string]interface{}, infile cafs.File) (skylark.StringDict, error) {
	m := &Module{ds: ds, secrets: secrets, infile: infile}
	st := skylarkstruct.FromStringDict(skylarkstruct.Default, skylark.StringDict{
		"set_meta":   skylark.NewBuiltin("set_meta", m.SetMeta),
		"get_config": skylark.NewBuiltin("get_config", m.GetConfig),
		"get_secret": skylark.NewBuiltin("get_secret", m.GetSecret),
		"get_body":   skylark.NewBuiltin("get_body", m.GetBody),
	})

	return skylark.StringDict{"qri": st}, nil
}

// Module encapsulates state for a qri skylark module
type Module struct {
	ds      *dataset.Dataset
	secrets map[string]interface{}
	infile  cafs.File
}

// GetConfig returns transformation configuration details
// TODO - supplying a string argument to qri.get_config('foo') should return the single config value instead of the whole map
func (m *Module) GetConfig(thread *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	if m.ds.Transform.Config == nil {
		return skylark.None, nil
	}
	return lib.Marshal(m.ds.Transform.Config)
}

// GetBody returns the body of the dataset we're transforming
func (m *Module) GetBody(thread *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	if m.infile == nil {
		return skylark.None, fmt.Errorf("qri.get_body failed: no DataFile")
	}
	rr, err := dsio.NewEntryReader(m.ds.Structure, m.infile)
	if err != nil {
		return skylark.None, fmt.Errorf("error allocating data reader: %s", err)
	}
	w, err := NewSkylarkEntryWriter(m.ds.Structure)
	if err != nil {
		return skylark.None, fmt.Errorf("error allocating skylark entry writer: %s", err)
	}

	err = dsio.Copy(rr, w)
	if err != nil {
		return skylark.None, err
	}
	return w.Value(), nil
}

// SetMeta sets a dataset meta field
func (m *Module) SetMeta(thread *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var keyx, valx skylark.Value
	if err := skylark.UnpackPositionalArgs("set_meta", args, kwargs, 2, &keyx, &valx); err != nil {
		return nil, err
	}

	if keyx.Type() != "string" {
		return nil, fmt.Errorf("expected key to be a string")
	}

	key, err := lib.AsString(keyx)
	if err != nil {
		return nil, fmt.Errorf("parsing string key: %s", err.Error())
	}

	val, err := lib.Unmarshal(valx)
	if err != nil {
		return nil, err
	}

	return skylark.None, m.ds.Meta.Set(key, val)
}

// GetSecret fetches a dict of secrets
// TODO - supplying a string argument to qri.get_secret('foo') should return the single secret value instead of the whole map
func (m *Module) GetSecret(thread *skylark.Thread, _ *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	if m.secrets == nil {
		return skylark.None, nil
	}

	return lib.Marshal(m.secrets)
}
