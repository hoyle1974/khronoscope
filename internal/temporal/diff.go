package temporal

import (
	"github.com/gabstv/go-bsdiff/pkg/bsdiff"
	"github.com/gabstv/go-bsdiff/pkg/bspatch"
)

type Diff []byte

func generateDiff(a, b []byte) (Diff, error) {
	patch, err := bsdiff.Bytes(a, b)
	if err != nil {
		return Diff{}, err
	}

	/*
		var buf bytes.Buffer
		patchWriter := zlib.NewWriter(&buf)
		patchWriter.Write(patch)
		patchWriter.Close()
		return buf.Bytes(), nil
	*/
	return patch, nil
}

func applyDiff(a []byte, diffData Diff) ([]byte, error) {
	// var out bytes.Buffer
	// r, _ := zlib.NewReader(bytes.NewReader(diffData))
	// io.Copy(&out, r)
	// r.Close()

	newfile, err := bspatch.Bytes(a, diffData) //out.Bytes())
	if err != nil {
		panic(err)
	}

	return newfile, nil
}
