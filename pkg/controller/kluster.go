package controller

import (
	"log"
	"time"

	klientset "github.com/viveksinghggits/kluster/pkg/client/clientset/versioned"
	kinf "github.com/viveksinghggits/kluster/pkg/client/informers/externalversions/viveksingh.dev/v1alpha1"
	klister "github.com/viveksinghggits/kluster/pkg/client/listers/viveksingh.dev/v1alpha1"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	// clientset for custom resource kluster
	klient klientset.Interface
	// kluster has synced
	klusterSynced cache.InformerSynced
	// lister
	kLister klister.KlusterLister
	// queue
	wq workqueue.RateLimitingInterface
}

func NewController(klient klientset.Interface, klusterInformer kinf.KlusterInformer) *Controller {
	c := &Controller{
		klient:        klient,
		klusterSynced: klusterInformer.Informer().HasSynced,
		kLister:       klusterInformer.Lister(),
		wq:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "kluster"),
	}

	klusterInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			DeleteFunc: c.handleDel,
		},
	)

	return c
}

func (c *Controller) Run(ch chan struct{}) error {
	if ok := cache.WaitForCacheSync(ch, c.klusterSynced); !ok {
		log.Println("cache was not sycned")
	}

	go wait.Until(c.worker, time.Second, ch)

	<-ch
	return nil
}

func (c *Controller) worker() {
	for c.processNextItem() {

	}
}

func (c *Controller) processNextItem() bool {
	item, shutDown := c.wq.Get()
	if shutDown {
		// logs as well
		return false
	}

	defer c.wq.Forget(item)
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		log.Printf("error %s calling Namespace key func on cache for item", err.Error())
		return false
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Printf("splitting key into namespace and name, error %s\n", err.Error())
		return false
	}

	kluster, err := c.kLister.Klusters(ns).Get(name)
	if err != nil {
		log.Printf("error %s, Getting the kluster resource from lister", err.Error())
		return false
	}
	log.Printf("kluster spec that we have is %+v\n", kluster.Spec)

	return true
}

func (c *Controller) handleAdd(obj interface{}) {
	log.Println("handleAdd was called")
	c.wq.Add(obj)
}

func (c *Controller) handleDel(obj interface{}) {
	log.Println("handleDel was called")
	c.wq.Add(obj)
}
