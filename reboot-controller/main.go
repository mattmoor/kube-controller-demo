package main

import (
	"flag"
	"fmt"
	"reflect"
	"time"

	"github.com/aaronlevy/kube-controller-demo/common"
	crv1 "github.com/aaronlevy/kube-controller-demo/apis/cr/v1"
	"github.com/golang/glog"
	// meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	// "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	// "k8s.io/apimachinery/pkg/watch"
	// "k8s.io/client-go/kubernetes"
	// lister_v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// TODO(aaron): make configurable and add MinAvailable
const maxUnavailable = 1

func NewClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := crv1.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}

	config := *cfg
	config.GroupVersion = &crv1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}

	return client, scheme, nil
}

func main() {
	// When running as a pod in-cluster, a kubeconfig is not needed. Instead this will make use of the service account injected into the pod.
	// However, allow the use of a local kubeconfig as this can make local development & testing easier.
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig file")

	// We log to stderr because glog will default to logging to a file.
	// By setting this debugging is easier via `kubectl logs`
	flag.Set("logtostderr", "true")
	flag.Parse()

	// Build the client config - optionally using a provided kubeconfig file.
	config, err := common.GetClientConfig(*kubeconfig)
	if err != nil {
		glog.Fatalf("Failed to load client config: %v", err)
	}

	// Construct the Kubernetes client
	client, _, err := NewClient(config)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes client: %v", err)
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	newPackageController(client).Run(stopCh)
}

type packageController struct {
	client     *rest.RESTClient
	informer   cache.Controller
	queue      workqueue.RateLimitingInterface
}

func newPackageController(client *rest.RESTClient) *packageController {
	rc := &packageController{
		client: client,
		queue:  workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	_, informer := cache.NewInformer(
		cache.NewListWatchFromClient(
			client,
			crv1.PackageResourcePlural,
			v1.NamespaceAll,
			fields.Everything()),
		// The types of objects this informer will return
		&crv1.Package{},
		// The resync period of this object. This will force a re-queue of all cached objects at this interval.
		// Every object will trigger the `Updatefunc` even if there have been no actual updates triggered.
		// In some cases you can set this to a very high interval - as you can assume you will see periodic updates in normal operation.
		// The interval is set low here for demo purposes.
		10*time.Second,
		// Callback Functions to trigger on add/update/delete
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				// rc.queue.Add(obj)
				glog.Infof("Add: %v", obj)
			},
			UpdateFunc: func(old, new interface{}) {
				// rc.queue.Add(new)
				if !reflect.DeepEqual(old, new) {
					glog.Infof("Update before: %v, after: %v", old, new)
				}
			},
			DeleteFunc: func(obj interface{}) {
				// rc.queue.Add(obj)
				glog.Infof("Delete: %v", obj)
			},
		},
	)

	rc.informer = informer

	return rc
}

func (c *packageController) Run(stopCh chan struct{}) {
	defer c.queue.ShutDown()
	glog.Info("Starting Package Controller")

	go c.informer.Run(stopCh)

	// Wait for all caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		glog.Error(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	// Launching additional goroutines would parallelize workers consuming from the queue (but we don't really need this)
	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
	glog.Info("Stopping Package Controller")
}

func (c *packageController) runWorker() {
	for c.processNext() {
	}
}

func (c *packageController) processNext() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.queue.Done(key)
	// Invoke the method containing the business logic
	err := c.process(key.(*v1.Pod))
	// Handle the error if something went wrong during the execution of the business logic
	c.handleErr(err, key)
	return true
}

func (c *packageController) process(pod *v1.Pod) error {
	glog.Infof("Received update of pod: %s", pod.GetName())
	return nil
}

func (c *packageController) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		glog.Infof("Error processing pod %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	glog.Errorf("Dropping pod %q out of the queue: %v", key, err)
}
