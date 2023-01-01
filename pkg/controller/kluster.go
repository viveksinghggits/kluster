package controller

import (
	"context"
	"log"
	"time"

	"github.com/viveksinghggits/kluster/pkg/apis/viveksingh.dev/v1alpha1"
	klientset "github.com/viveksinghggits/kluster/pkg/client/clientset/versioned"
	customscheme "github.com/viveksinghggits/kluster/pkg/client/clientset/versioned/scheme"
	kinf "github.com/viveksinghggits/kluster/pkg/client/informers/externalversions/viveksingh.dev/v1alpha1"
	klister "github.com/viveksinghggits/kluster/pkg/client/listers/viveksingh.dev/v1alpha1"
	"github.com/viveksinghggits/kluster/pkg/do"

	"github.com/kanisterio/kanister/pkg/poll"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const (
	klusterFinalizer = "viveksingh.dev/prod-protection"
	protectedNS      = "prod"
)

type Controller struct {
	client kubernetes.Interface

	// clientset for custom resource kluster
	klient klientset.Interface
	// kluster has synced
	klusterSynced cache.InformerSynced
	// lister
	kLister klister.KlusterLister
	// queue
	wq workqueue.RateLimitingInterface

	recorder record.EventRecorder
}

func NewController(client kubernetes.Interface, klient klientset.Interface, klusterInformer kinf.KlusterInformer) *Controller {
	runtime.Must(customscheme.AddToScheme(scheme.Scheme))

	eveBroadCaster := record.NewBroadcaster()
	eveBroadCaster.StartStructuredLogging(0)
	eveBroadCaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: client.CoreV1().Events(""),
	})
	recorder := eveBroadCaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "Kluster"})

	c := &Controller{
		client:        client,
		klient:        klient,
		klusterSynced: klusterInformer.Informer().HasSynced,
		kLister:       klusterInformer.Lister(),
		wq:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "kluster"),
		recorder:      recorder,
	}

	klusterInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			DeleteFunc: c.handleDel,
			UpdateFunc: c.handleUpdate,
		},
	)

	return c
}

// its gong to get called, whenever the resource is updated
func (c *Controller) handleUpdate(ondObj, newObj interface{}) {
	// get the kluster resource
	kluster, ok := newObj.(*v1alpha1.Kluster)
	if !ok {
		log.Printf("can not convert newObj to kluster resource\n")
		return
	}
	ctx := context.Background()
	// if the finalizer is set or not
	// check if the cluster has prod namespace
	_, err := c.client.CoreV1().Namespaces().Get(ctx, protectedNS, metav1.GetOptions{}) // this would requrie role change to be able to get ns
	if err == nil {
		// prod ns is available, do nothing
		return
	}
	// if it has, do nothing
	// otherwise, remove finalizer `viveksingh.dev/prod-protection` from resource
	// if we are here, there is an err set, to be explicit you can check this says resource not found
	k := kluster.DeepCopy()
	finals := []string{}
	for _, f := range k.Finalizers {
		if f == klusterFinalizer {
			continue
		}
		finals = append(finals, f)
	}
	k.Finalizers = finals

	// change role to be able to update the kluster resource
	if _, err = c.klient.ViveksinghV1alpha1().Klusters(k.Namespace).Update(ctx, k, metav1.UpdateOptions{}); err != nil {
		log.Printf("Update of the kluster resource failed: %s\n", err.Error())
		return
	}
	log.Println("Finalizer was removed from the resource")
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
		if apierrors.IsNotFound(err) {
			return deleteDOCluster()
		}

		log.Printf("error %s, Getting the kluster resource from lister", err.Error())
		return false
	}
	log.Printf("kluster spec that we have is %+v\n", kluster.Spec)

	clusterID, err := do.Create(c.client, kluster.Spec)
	if err != nil {
		// do something
		log.Printf("errro %s, creating the cluster", err.Error())
	}

	c.recorder.Event(kluster, corev1.EventTypeNormal, "ClusterCreation", "DO API was called to create the cluster")

	log.Printf("cluster id that we have is %s\n", clusterID)

	err = c.updateStatus(clusterID, "creating", kluster)
	if err != nil {
		log.Printf("error %s, updating status of the kluster %s\n", err.Error(), kluster.Name)
	}

	// query DO API to make sure clsuter' state is running
	err = c.waitForCluster(kluster.Spec, clusterID)
	if err != nil {
		log.Printf("error %s, waiting for cluster to be running", err.Error())
	}

	err = c.updateStatus(clusterID, "running", kluster)
	if err != nil {
		log.Printf("error %s updaring cluster status after waiting for cluster", err.Error())
	}

	c.recorder.Event(kluster, corev1.EventTypeNormal, "ClusterCreationCompleted", "DO Cluster creation was completed")
	return true
}

func deleteDOCluster() bool {
	// this actualy deletes the cluster from the DO
	log.Println("Cluster was deleted succcessfully")
	return true
}

func (c *Controller) waitForCluster(spec v1alpha1.KlusterSpec, clusterID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		state, err := do.ClusterState(c.client, spec, clusterID)
		if err != nil {
			return false, err
		}
		if state == "running" {
			return true, nil
		}

		return false, nil
	})
}

func (c *Controller) updateStatus(id, progress string, kluster *v1alpha1.Kluster) error {
	// get the latest version of kluster
	k, err := c.klient.ViveksinghV1alpha1().Klusters(kluster.Namespace).Get(context.Background(), kluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	k.Status.KlusterID = id
	k.Status.Progress = progress
	_, err = c.klient.ViveksinghV1alpha1().Klusters(kluster.Namespace).UpdateStatus(context.Background(), k, metav1.UpdateOptions{})
	return err
}

func (c *Controller) handleAdd(obj interface{}) {
	log.Println("handleAdd was called")
	c.wq.Add(obj)
}

func (c *Controller) handleDel(obj interface{}) {
	log.Println("handleDel was called")
	c.wq.Add(obj)
}
