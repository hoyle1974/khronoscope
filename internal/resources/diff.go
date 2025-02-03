package resources

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"sort"

	"github.com/gabstv/go-bsdiff/pkg/bsdiff"
	"github.com/gabstv/go-bsdiff/pkg/bspatch"
)

// **Constructs a suffix array for fast matching**
func SuffixArray(data []byte) []int {
	n := len(data)
	sa := make([]int, n)
	for i := 0; i < n; i++ {
		sa[i] = i
	}
	sort.Slice(sa, func(i, j int) bool {
		return bytes.Compare(data[sa[i]:], data[sa[j]:]) < 0
	})
	return sa
}

// **Finds the longest match using the suffix array**
func LongestMatch(a []byte, sa []int, b []byte, bOffset int) (int, int) {
	bestLen, bestPos := 0, -1
	low, high := 0, len(sa)

	for low < high {
		mid := (low + high) / 2
		pos := sa[mid]

		matchLen := 0
		for matchLen < len(b)-bOffset && pos+matchLen < len(a) && a[pos+matchLen] == b[bOffset+matchLen] {
			matchLen++
		}

		if matchLen > bestLen {
			bestLen, bestPos = matchLen, pos
		}

		if pos+matchLen < len(a) && bOffset+matchLen < len(b) {
			if a[pos+matchLen] < b[bOffset+matchLen] {
				low = mid + 1
			} else {
				high = mid
			}
		} else {
			break
		}
	}

	return bestPos, bestLen
}

type Diff2 []byte

func GenerateDiff2(a, b []byte) (Diff2, error) {
	patch, err := bsdiff.Bytes(a, b)
	if err != nil {
		return Diff2{}, err
	}

	var buf bytes.Buffer
	patchWriter := zlib.NewWriter(&buf)
	patchWriter.Write(patch)
	patchWriter.Close()
	return buf.Bytes(), nil
}

func ApplyDiff2(a []byte, diffData Diff2) ([]byte, error) {
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

type Diff []byte

// **Generates an efficient binary diff**
func GenerateDiff(a, b []byte) (Diff, error) {
	sa := SuffixArray(a)

	var buf bytes.Buffer
	patchWriter := zlib.NewWriter(&buf)

	bOffset := 0
	for bOffset < len(b) {
		matchPos, matchLen := LongestMatch(a, sa, b, bOffset)

		// If no match is found, store the new data
		if matchLen < 4 { // Avoid small unnecessary matches
			matchPos = -1
			matchLen = 0
		}

		// Encode the match position and length
		binary.Write(patchWriter, binary.LittleEndian, int32(matchPos))
		binary.Write(patchWriter, binary.LittleEndian, int32(matchLen))

		// Store new data if not matched
		newData := b[bOffset+matchLen:]
		binary.Write(patchWriter, binary.LittleEndian, int32(len(newData)))
		patchWriter.Write(newData)

		// Move forward
		bOffset += matchLen + len(newData)
	}

	patchWriter.Close()
	return buf.Bytes(), nil
}

// **Applies a binary diff to reconstruct B from A**
func ApplyDiff(a []byte, diffData Diff) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(diffData))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var b bytes.Buffer
	for {
		var matchPos, matchLen, newDataLen int32
		if err := binary.Read(reader, binary.LittleEndian, &matchPos); err != nil {
			break
		}
		binary.Read(reader, binary.LittleEndian, &matchLen)
		binary.Read(reader, binary.LittleEndian, &newDataLen)

		if matchPos >= 0 && matchLen > 0 {
			b.Write(a[matchPos : matchPos+matchLen])
		}

		newData := make([]byte, newDataLen)
		io.ReadFull(reader, newData)
		b.Write(newData)
	}

	return b.Bytes(), nil
}
