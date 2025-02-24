package access

import (
	"context"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/types"
	"github.com/muesli/cache2go"
	"github.com/rs/zerolog/log"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AccessController struct {
	client  conn.KhronosConn
	cache   *cache2go.CacheTable
	counter atomic.Int32
}

const ( // iota is reset to 0
	AccessNo    = iota // c0 == 0
	AccessOk    = iota // c1 == 1
	AccessMaybe = iota // c2 == 2
)

func NewAccessController(client conn.KhronosConn) *AccessController {
	return &AccessController{client: client, cache: cache2go.Cache("access")}
}

func (c *AccessController) CanViewResource(resource types.Resource) (int, error) {
	key := resource.GetKind() + "://" + resource.GetNamespace() + "/" + resource.GetName()
	item, err := c.cache.Value(key)
	if err == nil && item != nil {
		return item.Data().(int), nil
	}

	go func(key string) {
		c.counter.Add(1)
		defer log.Debug().Any("counter", c.counter.Add(-1)).Msg("Outstanding Access Requests")

		// Construct the resource attributes for the SAR
		resourceAttributes := authorizationv1.ResourceAttributes{
			Namespace: resource.GetNamespace(),
			Verb:      "get",              // You can change this to any verb (e.g., "list", "update", etc.)
			Resource:  resource.GetKind(), // Resource kind like "pods", "services", etc.
			Name:      resource.GetName(), // Name of the resource
		}

		// Check if the user has access
		_, err = c.client.Client.AuthorizationV1().SubjectAccessReviews().Create(context.Background(), &authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User:               c.client.CurrentUser,
				ResourceAttributes: &resourceAttributes,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			r := rand.Int31n(30) + 30
			c.cache.Add(key, time.Second*time.Duration(r), AccessNo)
			return
		}

		// If no error, access is allowed
		r := rand.Int31n(30) + 30
		c.cache.Add(key, time.Second*time.Duration(r), AccessOk)
	}(key)

	//return AccessMaybe, nil
	return AccessOk, nil
}
