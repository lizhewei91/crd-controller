package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	clientset "crd-controller/pkg/generated/clientset/versioned"
	crdscheme "crd-controller/pkg/generated/clientset/versioned/scheme"
	informers "crd-controller/pkg/generated/informers/externalversions/extension/v1"
	listers "crd-controller/pkg/generated/listers/extension/v1"
)

const controllerAgentName = "crd-controller"

type Controller struct {
	kubeclientset          kubernetes.Interface
	crdclientset           clientset.Interface
	deploymentlister       appslisters.DeploymentLister
	deploymentsynced       cache.InformerSynced
	uniteddeploymentlister listers.UnitedDeploymentLister
	uniteddeploymentsynced cache.InformerSynced
	workqueue              workqueue.RateLimitingInterface
	recorder               record.EventRecorder
}

func NewController(
	ctx context.Context,
	kubeClientset kubernetes.Interface,
	crdClientset clientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	unitedDeploymentInformer informers.UnitedDeploymentInformer,
) *Controller {
	logger := klog.FromContext(ctx)

	// 1. 向全局scheme注册crd资源
	utilruntime.Must(crdscheme.AddToScheme(scheme.Scheme))

	// 2. 创建recorder，记录event事件
	logger.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:          kubeClientset,
		crdclientset:           crdClientset,
		deploymentlister:       deploymentInformer.Lister(),
		deploymentsynced:       deploymentInformer.Informer().HasSynced,
		uniteddeploymentlister: unitedDeploymentInformer.Lister(),
		uniteddeploymentsynced: unitedDeploymentInformer.Informer().HasSynced,
		// 3.初始化workqueue
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "unitedDeployment"),
		recorder:  recorder,
	}

	logger.Info("Setting up event handlers")

	// 4. 给unitedDeployment的 informer增加回调函数
	unitedDeploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueUnitedDeployment,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueUnitedDeployment(new)
		},
	})

	// 5. 给deployment的 informer增加回调函数
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.Deployment)
			oldDepl := old.(*appsv1.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	return controller
}

func (c *Controller) enqueueUnitedDeployment(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	logger := klog.FromContext(context.Background())
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		logger.V(4).Info("Recovered deleted object", "resourceName", object.GetName())
	}
	logger.V(4).Info("Processing object", "object", klog.KObj(object))
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Foo, we should not do anything more
		// with it.
		if ownerRef.Kind != "UnitedDeployment" {
			return
		}

		ud, err := c.uniteddeploymentlister.UnitedDeployments(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			logger.V(4).Info("Ignore orphaned object", "object", klog.KObj(object), "foo", ownerRef.Name)
			return
		}

		c.enqueueUnitedDeployment(ud)
		return
	}
}

func (c *Controller) Run(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	logger := klog.FromContext(ctx)

	logger.Info("Starting unitedDeployment controller")

	logger.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(ctx.Done(), c.deploymentsynced, c.uniteddeploymentsynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logger.Info("Starting workers", "count", workers)

	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}
	logger.Info("Started workers")
	<-ctx.Done()
	logger.Info("Shutting down workers")
	return nil
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {

		//TODO controller logic

		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}
