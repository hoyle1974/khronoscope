package dao_test

import (
	"encoding/gob"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/hoyle1974/khronoscope/internal/dao"
	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/hoyle1974/khronoscope/internal/serializable"
)

func randomStrings(r *rand.Rand) []string {
	ret := []string{}

	for t := 0; t < 5; t++ {
		if r.IntN(2) == 0 {
			ret = append(ret, fmt.Sprintf("RandomString:%d", t))
		}
	}

	return ret
}

func randomMetrics(r *rand.Rand) map[string]resources.PodMetric {
	ret := map[string]resources.PodMetric{}
	ret["A"] = resources.PodMetric{CPUPercentage: r.Float64(), MemoryPercentage: r.Float64()}
	ret["B"] = resources.PodMetric{CPUPercentage: r.Float64(), MemoryPercentage: r.Float64()}
	ret["C"] = resources.PodMetric{CPUPercentage: r.Float64(), MemoryPercentage: r.Float64()}
	ret["D"] = resources.PodMetric{CPUPercentage: r.Float64(), MemoryPercentage: r.Float64()}
	if r.Float32() > .5 {
		ret["E"] = resources.PodMetric{CPUPercentage: r.Float64(), MemoryPercentage: r.Float64()}
	}
	return ret
}

func randomContainerInfo(r *rand.Rand) map[string]resources.ContainerInfo {
	ret := map[string]resources.ContainerInfo{}
	ret["A"] = resources.ContainerInfo{Image: newUuid(r), CPULimit: r.Int64(), MemoryLimit: r.Int64()}
	ret["B"] = resources.ContainerInfo{Image: newUuid(r), CPULimit: r.Int64(), MemoryLimit: r.Int64()}
	ret["C"] = resources.ContainerInfo{Image: newUuid(r), CPULimit: r.Int64(), MemoryLimit: r.Int64()}
	ret["D"] = resources.ContainerInfo{Image: newUuid(r), CPULimit: r.Int64(), MemoryLimit: r.Int64()}
	if r.Float32() > .5 {
		ret["E"] = resources.ContainerInfo{Image: newUuid(r), CPULimit: r.Int64(), MemoryLimit: r.Int64()}
	}
	return ret
}

func newUuid(r *rand.Rand) string {
	return fmt.Sprintf("UUID:%v", r.Int64())
}

func randomPod(r *rand.Rand) resources.Resource {
	extra := resources.PodExtra{
		Phase:       newUuid(r),
		Node:        newUuid(r),
		Metrics:     randomMetrics(r),
		Uptime:      time.Duration(r.IntN(1000)),
		StartTime:   serializable.Time{Time: time.Now()},
		Containers:  randomContainerInfo(r),
		Labels:      randomStrings(r),
		Annotations: randomStrings(r),
		Logs:        randomStrings(r),
		Logging:     r.Float32() > .5,
	}
	return resources.Resource{
		Uid:       "PodUid",
		Timestamp: serializable.Time{Time: time.Now()},
		Kind:      "Pod",
		Namespace: "Default",
		Name:      "PodName",
		Extra:     extra,
		Details:   randomStrings(r),
	}
}

func randomResource(r *rand.Rand) resources.Resource {
	switch r.IntN(1) {
	case 0:
		return randomPod(r)
	}

	return randomPod(r)
}

func Test_Store(t *testing.T) {
	gob.Register(resources.Resource{})
	gob.Register(resources.PodExtra{})

	store := dao.New()
	resources.RegisterResourceRenderer("Pod", resources.PodRenderer{})

	a := rand.Uint64()
	r := rand.New(rand.NewPCG(0, a))

	times := []time.Time{}
	data := []resources.Resource{}
	for i := 0; i < 100; i++ {
		resource := randomResource(r)
		// fmt.Printf("%s\n", strings.Join(resource.GetDetails(), "\n"))

		times = append(times, resource.GetTimestamp())
		data = append(data, resource)

		store.AddResource(resource)
	}

	// fmt.Println("-----------------------------")
	for i := 0; i < len(times); i++ {
		m := store.GetResourcesAt(times[i], "", "")

		if cmp.Equal(data[i], m[0], cmpopts.EquateEmpty()) {
			// fmt.Printf("%s\n", strings.Join(m[0].GetDetails(), "\n"))
			t.Fatalf("Seed=%v, Mismatch(%d)!\n", a, i)
		}
	}

	store.Save("test.dat")

}
