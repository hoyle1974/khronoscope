package dao

import (
	"context"
	"sync"
	"time"

	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/rs/zerolog/log"

	temp "github.com/hoyle1974/temporal"
)

type storeImpl struct {
	lock    sync.Mutex
	storage temp.ReadWriteMap
}

// func (s *storeImpl) tick() {
// 	ticker := time.NewTicker(1 * time.Second)

// 	t := time.Now().Truncate(time.Second)
// 	for range ticker.C {
// 		s.lock.Lock()
// 		temp := misc.DeepCopyArray(s.resources)
// 		s.lock.Unlock()

// 		data, _ := misc.EncodeToBytes(temp)
// 		s.storageManager.WriteData(context.TODO(), t, data)
// 		t = t.Add(1 * time.Second)
// 	}
// }

func (s *storeImpl) GetResourcesAt(timestamp time.Time, kind string, namespace string) []resources.Resource {
	// maxTime := s.storage.GetMaxTime()

	s.lock.Lock()
	defer s.lock.Unlock()

	var ret []resources.Resource

	// if timestamp.After(maxTime) {
	// 	ret = s.resources
	// } else {
	data, err := s.storage.GetAll(context.TODO(), timestamp)
	if err != nil {
		return []resources.Resource{}
	}

	ret = make([]resources.Resource, 0, len(data))
	for _, v := range data {
		var res resources.Resource
		if misc.DecodeFromBytes(v, &res) != nil {
			return []resources.Resource{}
		}
		ret = append(ret, res)
	}
	// }

	ret = misc.Filter(ret, func(r resources.Resource) bool {
		if kind != "" && kind != r.Kind {
			return false
		}
		if namespace != "" && namespace != r.Namespace {
			return false
		}
		return true
	})

	return ret
}

func (s *storeImpl) GetResourceAt(timestamp time.Time, uid string) (resources.Resource, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	data, err := s.storage.GetAll(context.TODO(), timestamp)
	if err != nil {
		return resources.Resource{}, err
	}

	for _, v := range data {
		var res resources.Resource
		if misc.DecodeFromBytes(v, &res) != nil {
			return resources.Resource{}, err
		}
		if res.Uid == uid {
			return res, nil
		}
	}

	return resources.Resource{}, nil
}

func (s *storeImpl) GetTimeRange() (time.Time, time.Time) {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.storage.GetMinMaxTime()
}

func (s *storeImpl) AddResource(resource resources.Resource) {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Debug().Str("kind", resource.GetKind()).Str("name", resource.GetName()).Str("ns", resource.GetNamespace()).Msg("AddResource")

	data, err := misc.EncodeToBytes(resource)
	if err != nil {
		return
	}
	err = s.storage.Set(context.TODO(), resource.GetTimestamp(), resource.GetUID(), data)
	if err != nil {
		return
	}
}

func (s *storeImpl) UpdateResource(resource resources.Resource) {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Debug().Str("kind", resource.GetKind()).Str("name", resource.GetName()).Str("ns", resource.GetNamespace()).Msg("UpdateResource")

	data, err := misc.EncodeToBytes(resource)
	if err != nil {
		return
	}
	err = s.storage.Set(context.TODO(), resource.GetTimestamp(), resource.GetUID(), data)
	if err != nil {
		return
	}
}

func (s *storeImpl) DeleteResource(resource resources.Resource) {
	s.lock.Lock()
	defer s.lock.Unlock()
	log.Debug().Str("kind", resource.GetKind()).Str("name", resource.GetName()).Str("ns", resource.GetNamespace()).Msg("DeleteResource")
	err := s.storage.Del(context.TODO(), resource.GetTimestamp(), resource.GetUID())
	if err != nil {
		return
	}
}

// Not supported
func (s *storeImpl) SetLabel(time.Time, string) {
}

func (s *storeImpl) GetLabel(time time.Time) string {
	return ""
}

func (s *storeImpl) GetNextLabelTime(time.Time) time.Time {
	return time.Now()
}

func (s *storeImpl) GetPrevLabelTime(time.Time) time.Time {
	return time.Now()
}

func (s *storeImpl) Save(string) {
}

func (s *storeImpl) Size() int { return 0 }
