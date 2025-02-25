package dao_test

import (
	"encoding/gob"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/hoyle1974/khronoscope/internal/dao"
	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/hoyle1974/khronoscope/internal/serializable"
)

func randomPod(r *rand.Rand) resources.Resource {
	extra := map[string]any{}
	return resources.Resource{
		Uid:       "PodUid",
		Timestamp: serializable.Time{Time: time.Now()},
		Kind:      "Pod",
		Namespace: "Default",
		Name:      "PodName",
		Extra:     extra,
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
