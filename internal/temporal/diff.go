package temporal

import (
	"bytes"
	"compress/zlib"
	"encoding/gob"
	"io"

	"github.com/gabstv/go-bsdiff/pkg/bsdiff"
	"github.com/gabstv/go-bsdiff/pkg/bspatch"
)

type Diff []byte

func EncodeToBytes(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func generateDiff(a, b []byte) (Diff, error) {
	patch, err := bsdiff.Bytes(a, b)
	if err != nil {
		return Diff{}, err
	}

	var buf bytes.Buffer
	patchWriter := zlib.NewWriter(&buf)
	patchWriter.Write(patch)
	patchWriter.Close()
	return buf.Bytes(), nil
}

func applyDiff(a []byte, diffData Diff) ([]byte, error) {
	var out bytes.Buffer
	r, _ := zlib.NewReader(bytes.NewReader(diffData))
	io.Copy(&out, r)
	r.Close()

	newfile, err := bspatch.Bytes(a, out.Bytes())
	if err != nil {
		panic(err)
	}

	return newfile, nil
}
