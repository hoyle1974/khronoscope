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

	return patch, nil
}

func applyDiff(a []byte, diffData Diff) ([]byte, error) {
	newfile, err := bspatch.Bytes(a, diffData)
	if err != nil {
		panic(err)
	}

	return newfile, nil
}
