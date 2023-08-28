/*
Copyright 2023 The Kubernetes crd-controller Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"

	"crd-controller/pkg/controller"
	clientset "crd-controller/pkg/generated/clientset/versioned"
	informers "crd-controller/pkg/generated/informers/externalversions"
	"crd-controller/pkg/signals"
)

var (
	kubeconfig string

	pprofAddr   string
	enablePprof bool

	leaseLockName      string
	leaseLockNamespace string
	leaseLockId        string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&pprofAddr, "pprof-addr", ":8090", "The address the pprof binds to.")
	flag.BoolVar(&enablePprof, "enable-pprof", false, "Enable pprof for crd-controller.")
	flag.StringVar(&leaseLockId, "lease-lock-id", fmt.Sprintf("crd-controller-%v", uuid.New().String()),
		"lease-lock-id determines the name of the resource that leader election will use for holding the leader lock, Default is crd-controller-uuid.")
	flag.StringVar(&leaseLockName, "lease-lock-name", "", "the lease lock resource name")
	flag.StringVar(&leaseLockNamespace, "lease-lock-namespace", "", "the lease lock resource namespace")
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	ctx := signals.SetupSignalHandler()
	logger := klog.FromContext(ctx)

	// 1、若需要，开启 pprof
	if enablePprof {
		go func() {
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				klog.Error(err, "unable to start pprof")
			}
		}()
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logger.Error(err, "Error building kubeconfig")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	// 2、创建kubeClient、crdClient客户端
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	crdClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	// 3、创建 kubeInformerFactory、crdInformerFactory
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	crdInformerFactory := informers.NewSharedInformerFactory(crdClient, time.Second*30)

	// 4、创建 controller 实例
	controller := controller.NewController(ctx, kubeClient, crdClient,
		kubeInformerFactory.Apps().V1().Deployments(),
		crdInformerFactory.Extension().V1().UnitedDeployments())

	// 5、运行 kubeInformerFactory、crdInformerFactory
	kubeInformerFactory.Start(ctx.Done())
	crdInformerFactory.Start(ctx.Done())

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseLockName,
			Namespace: leaseLockNamespace,
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: leaseLockId,
		},
	}

	// 6、进行选主逻辑
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock: lock,
		// IMPORTANT: you MUST ensure that any code you have that
		// is protected by the lease must terminate **before**
		// you call cancel. Otherwise, you could have a background
		// loop still running and another process could
		// get elected before your background loop finished, violating
		// the stated goal of the lease.
		ReleaseOnCancel: true,
		LeaseDuration:   60 * time.Second,
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {

				// 7、选主成功，运行 controller 逻辑
				if err = controller.Run(ctx, 2); err != nil {
					logger.Error(err, "Error running controller")
					klog.FlushAndExit(klog.ExitFlushTimeout, 1)
				}
			},
			OnStoppedLeading: func() {
				// we can do cleanup here
				klog.Infof("leader lost: %s", leaseLockId)
				os.Exit(0)
			},
			OnNewLeader: func(identity string) {
				// we're notified when new leader elected
				if identity == leaseLockId {
					// I just got the lock
					return
				}
				klog.Infof("new leader elected: %s", identity)
			},
		},
	})
}
