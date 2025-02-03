package resources

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hoyle1974/khronoscope/internal/serializable"
)

/*

Problem Statement.  We have a struct in resources defined as such:

type Resource struct {
	Uid       string            // The Uid of the k8s object
	Timestamp serializable.Time // The timestamp that this resource is valid for
	Kind      string            // The k8s kind of resource
	Namespace string            // The k8s namespace, may be empty for things like namespace and node resources
	Name      string            // The name of the resource
	Extra     Copyable          // This should be a custom, gob registered and serializable object if used
	Details   []string
}

In resources we also have:

type Copyable interface {
	Copy() Copyable
}

In our serializable package we have

type Time struct {
	Time     time.Time
	Location string
}

We do know all of the Extra types ahead of time.

The Extra field can be things like this:

type PodMetric struct {
	CPUPercentage    float64
	MemoryPercentage float64
}

type ContainerInfo struct {
	Image       string
	CPULimit    int64
	MemoryLimit int64
}

type PodExtra struct {
	Phase       string
	Node        string
	Metrics     map[string]PodMetric
	Uptime      time.Duration
	StartTime   time.Time
	Containers  map[string]ContainerInfo
	Labels      []string
	Annotations []string
	Logs        []string
	Logging     bool
}

or as simple as this:


type ReplicaSetExtra struct {
	Replicas             int32
	AvailableReplicas    int32
	ReadyReplicas        int32
	FullyLabeledReplicas int32
}

or this

type NodeExtra struct {
	Metrics               map[string]string
	NodeCreationTimestamp time.Time
	CPUCapacity           int64
	MemCapacity           int64
	Uptime                time.Duration
	PodMetrics            map[string]map[string]PodMetric
}



Resources and their data are immutable.

We currently store, in memory, a copy of a Resource each time it changes.  To conserve space I want to be able to store a diff of a resource with the previous resource, then be able to reconstruct the latest resource by applying the diffs in chronological order.

Resources and associated types are registered with GOB if that helps.  GOB is not required to solve this problem set.

Generate a function called GenerateDiff that takes resources A & B and returns a Diff object that is a binary diff of the two objects that is optimized for size.

There should also be a function ApplyDiff that takes resource A and the Diff previously generated and produce an object that is equivalent to resource B


*/

func Test_Diff2(t *testing.T) {
	a := []byte("Hello, this is version A of the document! Hello, this is version A of the document!")
	b := []byte("Hello, this is version B of the document! Hello, this is version A of the document! A")

	diff, err := GenerateDiff(a, b)
	if err != nil {
		fmt.Println("Error generating diff:", err)
		return
	}

	reconstructed, err := ApplyDiff(a, diff)
	if err != nil {
		fmt.Println("Error applying diff:", err)
		return
	}

	fmt.Println("Original A:   ", string(a))
	fmt.Println("Modified B:   ", string(b))
	fmt.Println("Reconstructed:", string(reconstructed))
	fmt.Printf("Len A: %d   Len B: %d\n", len(a), len(b))
	fmt.Println("Diff Size:    ", len(diff))
}

func EncodeToBytes(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Test_Diff(t *testing.T) {
	gob.Register(Resource{})
	gob.Register(PodExtra{})
	gob.Register(serializable.Time{})

	a := Resource{
		Uid:       "123",
		Name:      "Me",
		Timestamp: serializable.Time{Time: time.Now()},
		Kind:      "Pod",
		Namespace: "Foo",
		Details:   []string{"tag1", "tag2"},
		Extra:     PodExtra{Phase: "Hello", Node: "99"},
	}

	b := Resource{
		Uid:       "123",
		Name:      "Me",
		Timestamp: serializable.Time{Time: time.Now()},
		Kind:      "Pod",
		Namespace: "Foo",
		Details:   []string{"tag1", "tag2", "tag3"},
		Extra:     PodExtra{Phase: "Hello Yall", Node: "992", Logging: true, Logs: []string{"a", "b", "c"}},
	}
	b = a

	aa, _ := EncodeToBytes(a)
	bb, _ := EncodeToBytes(b)

	/*
		patch, _ := bsdiff.Bytes(aa, bb)

		fmt.Printf("Len A: %d   Len B: %d\n", len(aa), len(bb))
		fmt.Println("Diff Size:    ", len(patch))

		var buf bytes.Buffer
		patchWriter := zlib.NewWriter(&buf)
		patchWriter.Write(patch)
		patchWriter.Close()
		fmt.Println("Diff Size:    ", len(buf.Bytes()))
	*/
	diff2, err := GenerateDiff2(aa, bb)
	if err != nil {
		fmt.Println("Error generating diff:", err)
		return
	}
	if na, err := ApplyDiff2(aa, diff2); err != nil {
		fmt.Println("Error applying diff:", err)
		return
	} else {
		fmt.Println("Equal:", bytes.Compare(na, bb))
		// fmt.Println("Updated A to match B:", na)
		fmt.Printf("Len A: %d   Len B: %d\n", len(aa), len(bb))
		fmt.Println("Diff Size:    ", len(diff2))
		fmt.Println("Compare: ", bytes.Compare(bb, na))
	}

	diff, err := GenerateDiff(aa, bb)
	if err != nil {
		fmt.Println("Error generating diff:", err)
		return
	}

	// Apply diff to A
	if na, err := ApplyDiff(aa, diff); err != nil {
		fmt.Println("Error applying diff:", err)
		return
	} else {
		fmt.Println("Equal:", bytes.Compare(na, bb))
		// fmt.Println("Updated A to match B:", na)
		fmt.Printf("Len A: %d   Len B: %d\n", len(aa), len(bb))
		fmt.Println("Diff Size:    ", len(diff))
		fmt.Println("Compare: ", bytes.Compare(bb, na))
	}

}

func randomResource() Resource {
	k := []string{}
	for idx := 0; idx < rand.IntN(100); idx++ {
		k = append(k, uuid.New().String())
	}

	r := Resource{
		Uid:       uuid.New().String(),
		Name:      uuid.New().String(),
		Timestamp: serializable.Time{Time: time.Now()},
		Kind:      uuid.New().String(),
		Namespace: uuid.New().String(),
		Details:   k,
		Extra:     PodExtra{Phase: "Hello", Node: "99"},
	}

	return r
}
func mutateResource(r Resource) Resource {
	for idx := 0; idx < len(r.Details); idx++ {
		if rand.Float32() > .9 {
			r.Details[idx] = uuid.New().String()
		}
	}
	r.Timestamp = serializable.Time{Time: time.Now()}

	return r
}

func Test_RandomDiff(t *testing.T) {
	gob.Register(Resource{})
	gob.Register(PodExtra{})
	gob.Register(serializable.Time{})

	ca, cb, cc, cd := 0, 0, 0, 0
	a := randomResource()
	for tt := 0; tt < 100; tt++ {

		b := mutateResource(a)

		aa, _ := EncodeToBytes(a)
		bb, _ := EncodeToBytes(b)
		ca += len(aa)
		cb += len(bb)

		d, _ := GenerateDiff(aa, bb)
		cc += len(d)
		cc, _ := ApplyDiff(aa, d)
		if bytes.Compare(bb, cc) != 0 {
			t.Error("ApplyDiff failed")
		}

		d2, _ := GenerateDiff2(aa, bb)
		cd += len(d2)
		cc2, _ := ApplyDiff2(aa, d2)
		if bytes.Compare(bb, cc2) != 0 {
			t.Error("ApplyDiff failed")
		}

		a = b
	}
	fmt.Println(ca, cb, cc, cd)

}
