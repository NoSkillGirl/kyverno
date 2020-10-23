package policyreport

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	kyverno "github.com/kyverno/kyverno/pkg/api/kyverno/v1"
	policyreportclient "github.com/kyverno/kyverno/pkg/client/clientset/versioned"
	policyreportinformer "github.com/kyverno/kyverno/pkg/client/informers/externalversions/policyreport/v1alpha1"
	"github.com/kyverno/kyverno/pkg/config"
	"github.com/kyverno/kyverno/pkg/constant"
	dclient "github.com/kyverno/kyverno/pkg/dclient"
	"github.com/kyverno/kyverno/pkg/policyreport/jobs"
	"github.com/kyverno/kyverno/pkg/policystatus"
	v1 "k8s.io/api/core/v1"
	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const workQueueName = "policy-violation-controller"
const workQueueRetryLimit = 3

//Generator creates PV
type Generator struct {
	dclient *dclient.Client

	// returns true if the cluster policy store has been synced at least once
	prSynced cache.InformerSynced
	// returns true if the namespaced cluster policy store has been synced at at least once
	nsprSynced cache.InformerSynced
	log        logr.Logger
	queue      workqueue.RateLimitingInterface
	dataStore  *dataStore

	inMemoryConfigMap *PVEvent
	mux               sync.Mutex
	job               *jobs.Job
}

//NewDataStore returns an instance of data store
func newDataStore() *dataStore {
	ds := dataStore{
		data: make(map[string]Info),
	}
	return &ds
}

type dataStore struct {
	data map[string]Info
	mu   sync.RWMutex
}

func (ds *dataStore) add(keyHash string, info Info) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	// queue the key hash
	ds.data[keyHash] = info
}

func (ds *dataStore) lookup(keyHash string) Info {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.data[keyHash]
}

func (ds *dataStore) delete(keyHash string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.data, keyHash)
}

//Info is a request to create PV
type Info struct {
	PolicyName string
	Resource   unstructured.Unstructured
	Rules      []kyverno.ViolatedRule
	FromSync   bool
}

func (i Info) toKey() string {
	keys := []string{
		i.PolicyName,
		i.Resource.GetKind(),
		i.Resource.GetNamespace(),
		i.Resource.GetName(),
		strconv.Itoa(len(i.Rules)),
	}
	return strings.Join(keys, "/")
}

// make the struct hashable

type PVEvent struct {
	Namespace map[string][]Info
	Cluster   map[string][]Info
}

//GeneratorInterface provides API to create PVs
type GeneratorInterface interface {
	Add(infos ...Info)
}

// NewPRGenerator returns a new instance of policy violation generator
func NewPRGenerator(client *policyreportclient.Clientset,
	dclient *dclient.Client,
	prInformer policyreportinformer.ClusterPolicyReportInformer,
	nsprInformer policyreportinformer.PolicyReportInformer,
	policyStatus policystatus.Listener,
	job *jobs.Job,
	log logr.Logger) *Generator {
	gen := Generator{
		dclient:    dclient,
		prSynced:   prInformer.Informer().HasSynced,
		nsprSynced: nsprInformer.Informer().HasSynced,
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), workQueueName),
		dataStore:  newDataStore(),
		log:        log,
		inMemoryConfigMap: &PVEvent{
			Namespace: make(map[string][]Info),
			Cluster:   make(map[string][]Info),
		},
		job: job,
	}

	return &gen
}

func (gen *Generator) enqueue(info Info) {
	// add to data map
	keyHash := info.toKey()
	// add to
	// queue the key hash

	gen.dataStore.add(keyHash, info)
	gen.queue.Add(keyHash)
}

//Add queues a policy violation create request
func (gen *Generator) Add(infos ...Info) {
	for _, info := range infos {
		gen.enqueue(info)
	}
}

// Run starts the workers
func (gen *Generator) Run(workers int, stopCh <-chan struct{}) {
	logger := gen.log
	defer utilruntime.HandleCrash()
	logger.Info("start")
	defer logger.Info("shutting down")

	if !cache.WaitForCacheSync(stopCh, gen.prSynced, gen.nsprSynced) {
		logger.Info("failed to sync informer cache")
	}

	go func(gen *Generator) {
		ticker := time.Tick(constant.PolicyReportResourceChangeResync)
		for {
			select {
			case <-ticker:
				err := gen.createConfigmap()
				scops := []string{}
				if len(gen.inMemoryConfigMap.Namespace) > 0 {
					scops = append(scops, constant.Namespace)
				}
				if len(gen.inMemoryConfigMap.Cluster["cluster"]) > 0 {
					scops = append(scops, constant.Cluster)
				}
				gen.job.Add(jobs.JobInfo{
					JobType: constant.ConfigmapMode,
					JobData: strings.Join(scops, ","),
				})
				if err != nil {
					gen.log.Error(err, "configmap error")
				}
				gen.inMemoryConfigMap = &PVEvent{
					Namespace: make(map[string][]Info),
					Cluster:   make(map[string][]Info),
				}
			}
		}
	}(gen)

	for i := 0; i < workers; i++ {
		go wait.Until(gen.runWorker, constant.PolicyViolationControllerResync, stopCh)
	}

	<-stopCh
}

func (gen *Generator) runWorker() {
	for gen.processNextWorkItem() {
	}
}

func (gen *Generator) handleErr(err error, key interface{}) {
	logger := gen.log
	if err == nil {
		gen.queue.Forget(key)
		return
	}

	// retires requests if there is error
	if gen.queue.NumRequeues(key) < workQueueRetryLimit {
		logger.Error(err, "failed to sync policy violation", "key", key)
		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		gen.queue.AddRateLimited(key)
		return
	}
	gen.queue.Forget(key)
	// remove from data store
	if keyHash, ok := key.(string); ok {
		gen.dataStore.delete(keyHash)
	}
	logger.Error(err, "dropping key out of the queue", "key", key)
}

func (gen *Generator) processNextWorkItem() bool {
	logger := gen.log
	obj, shutdown := gen.queue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer gen.queue.Done(obj)
		var keyHash string
		var ok bool

		if keyHash, ok = obj.(string); !ok {
			gen.queue.Forget(obj)
			logger.Info("incorrect type; expecting type 'string'", "obj", obj)
			return nil
		}

		// lookup data store
		info := gen.dataStore.lookup(keyHash)
		if reflect.DeepEqual(info, Info{}) {
			// empty key
			gen.queue.Forget(obj)
			logger.Info("empty key")
			return nil
		}

		err := gen.syncHandler(info)
		gen.handleErr(err, obj)
		return nil
	}(obj)

	if err != nil {
		logger.Error(err, "failed to process item")
		return true
	}

	return true
}
func (gen *Generator) createConfigmap() error {
	defer func() {
		gen.mux.Unlock()
	}()
	gen.mux.Lock()
	configmap, err := gen.dclient.GetResource("", "ConfigMap", config.KubePolicyNamespace, "kyverno-event")
	if err != nil {
		return err
	}
	cm := v1.ConfigMap{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(configmap.UnstructuredContent(), &cm); err != nil {
		return err
	}

	rawData, _ := json.Marshal(gen.inMemoryConfigMap.Cluster)
	cm.Data[constant.Cluster] = string(rawData)
	rawData, _ = json.Marshal(gen.inMemoryConfigMap.Namespace)
	cm.Data[constant.Namespace] = string(rawData)

	_, err = gen.dclient.UpdateResource("", "ConfigMap", config.KubePolicyNamespace, cm, false)
	if err != nil {
		return err
	}
	return nil
}

func (gen *Generator) syncHandler(info Info) error {
	gen.mux.Lock()
	defer gen.mux.Unlock()

	if info.Resource.GetNamespace() == "" {
		// cluster scope resource generate a clusterpolicy violation
		gen.inMemoryConfigMap.Cluster["cluster"] = append(gen.inMemoryConfigMap.Cluster["cluster"], info)
		return nil
	}

	// namespaced resources generated a namespaced policy violation in the namespace of the resource
	if _, ok := gen.inMemoryConfigMap.Namespace[info.Resource.GetNamespace()]; ok {
		gen.inMemoryConfigMap.Namespace[info.Resource.GetNamespace()] = append(gen.inMemoryConfigMap.Namespace[info.Resource.GetNamespace()], info)
	}
	return nil
}