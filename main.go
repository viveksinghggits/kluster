package main

import (
	"flag"
	"log"
	"path/filepath"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	klient "github.com/viveksinghggits/kluster/pkg/client/clientset/versioned"
	kInfFac "github.com/viveksinghggits/kluster/pkg/client/informers/externalversions"
	"github.com/viveksinghggits/kluster/pkg/controller"
)

func main() {
	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Printf("Building config from flags, %s", err.Error())
	}

	klientset, err := klient.NewForConfig(config)
	if err != nil {
		log.Printf("getting klient set %s\n", err.Error())
	}

	infoFactory := kInfFac.NewSharedInformerFactory(klientset, 20*time.Minute)
	ch := make(chan struct{})
	c := controller.NewController(klientset, infoFactory.Viveksingh().V1alpha1().Klusters())

	infoFactory.Start(ch)
	if err := c.Run(ch); err != nil {
		log.Printf("error running controller %s\n", err.Error())
	}
}
