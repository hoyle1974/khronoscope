package conn

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

type KhronosConn struct {
	Client          kubernetes.Interface
	DynamicClient   dynamic.Interface
	MetricsClient   *metrics.Clientset
	Config          *rest.Config
	DiscoveryClient *discovery.DiscoveryClient
}

func NewKhronosConnection(kubeConfigFlag *string) (KhronosConn, error) {
	// Disable the kubernetes logger otherwise it will mess output up from time to time.
	klog.SetLogger(logr.Logger{})

	// Otherwise use passed in flags
	kubeConfigPath := *kubeConfigFlag

	// Look for KUBECONFIG and use it if it exists
	if kubeConfigPath == "" {
		kubeConfigPath = os.Getenv("KUBECONFIG")
	}

	// Use a default
	if kubeConfigPath == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeConfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	var kubeconfig *rest.Config
	if kubeConfigPath != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return KhronosConn{}, fmt.Errorf("unable to load kubeconfig from %s: %v", kubeConfigPath, err)
		}
		kubeconfig = config
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			return KhronosConn{}, fmt.Errorf("unable to load in-cluster config: %v", err)
		}
		kubeconfig = config
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return KhronosConn{}, fmt.Errorf("unable to create a client: %v", err)
	}

	// Create dynamic client to interact with any resource
	dynamicClient, err := dynamic.NewForConfig(kubeconfig)
	if err != nil {
		return KhronosConn{}, fmt.Errorf("unable to create dynamic client: %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(kubeconfig)
	if err != nil {
		panic(err)
	}

	mc, err := metrics.NewForConfig(kubeconfig)
	if err != nil {
		return KhronosConn{}, fmt.Errorf("unable to create a metric client: %v", err)
	}

	return KhronosConn{Client: client, DynamicClient: dynamicClient, DiscoveryClient: discoveryClient, MetricsClient: mc, Config: kubeconfig}, nil
}
