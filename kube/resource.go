package kube

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/c-bata/go-prompt"
	"k8s.io/api/core/v1"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps_v1 "k8s.io/api/apps/v1"
	ext_v1beta1 "k8s.io/api/extensions/v1beta1"
)

const thresholdFetchInterval = 10 * time.Second

var resourceTypes = []prompt.Suggest{
	{Text: "clusterroles"},
	{Text: "clusters"}, // valid only for federation apiservers
	{Text: "componentstatuses"},
	{Text: "configmaps"},
	{Text: "daemonsets"},
	{Text: "deployments"},
	{Text: "endpoints"},
	{Text: "events"},
	{Text: "horizontalpodautoscalers"},
	{Text: "ingresses"},
	{Text: "jobs"},
	{Text: "limitranges"},
	{Text: "namespaces"},
	{Text: "networkpolicies"},
	{Text: "nodes"},
	{Text: "persistentvolumeclaims"},
	{Text: "persistentvolumes"},
	{Text: "pod"},
	{Text: "podsecuritypolicies"},
	{Text: "podtemplates"},
	{Text: "replicasets"},
	{Text: "replicationcontrollers"},
	{Text: "resourcequotas"},
	{Text: "roles"},
	{Text: "secrets"},
	{Text: "serviceaccounts"},
	{Text: "services"},
	{Text: "statefulsets"},
	{Text: "storageclasses"},
	{Text: "thirdpartyresources"},

	// aliases
	{Text: "cs"},
	{Text: "cm"},
	{Text: "ds"},
	{Text: "deploy"},
	{Text: "ep"},
	{Text: "hpa"},
	{Text: "ing"},
	{Text: "limits"},
	{Text: "ns"},
	{Text: "no"},
	{Text: "pvc"},
	{Text: "pv"},
	{Text: "po"},
	{Text: "psp"},
	{Text: "rs"},
	{Text: "rc"},
	{Text: "quota"},
	{Text: "sa"},
	{Text: "svc"},
}

func init() {
	lastFetchedAt = new(sync.Map)
	podList = new(sync.Map)
	endpointList = new(sync.Map)
	deploymentList = new(sync.Map)
	daemonSetList = new(sync.Map)
	eventList = new(sync.Map)
	secretList = new(sync.Map)
	ingressList = new(sync.Map)
	limitRangeList = new(sync.Map)
	persistentVolumeClaimsList = new(sync.Map)
	podTemplateList = new(sync.Map)
	replicaSetList = new(sync.Map)
	replicationControllerList = new(sync.Map)
	resourceQuotaList = new(sync.Map)
	rolesList = new(sync.Map)
	serviceAccountList = new(sync.Map)
	serviceList = new(sync.Map)
}

/* LastFetchedAt */

var (
	lastFetchedAt *sync.Map
)

func shouldFetch(key string) bool {
	v, ok := lastFetchedAt.Load(key)
	if !ok {
		log.Printf("[WARN] Not found %s in lastFetchedAt", key)
		return true
	}
	t, ok := v.(time.Time)
	if !ok {
		return true
	}
	return time.Since(t) > thresholdFetchInterval
}

func updateLastFetchedAt(key string) {
	lastFetchedAt.Store(key, time.Now())
}

/* Cluster Role */

var (
	clusterRoleList atomic.Value
)

func fetchClusterRoleList() {
	key := "cluster_role"
	if !shouldFetch(key) {
		return
	}
	l, _ := getClient().RbacV1().ClusterRoles().List(meta_v1.ListOptions{})
	clusterRoleList.Store(l)
	updateLastFetchedAt(key)
}

func getClusterRoleSuggestions() []prompt.Suggest {
	go fetchClusterRoleList()
	l, ok := clusterRoleList.Load().(*rbac_v1.ClusterRoleList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Component Status */

var (
	componentStatusList atomic.Value
)

func fetchComponentStatusList() {
	key := "component_status"
	if !shouldFetch(key) {
		return
	}
	l, _ := getClient().CoreV1().ComponentStatuses().List(meta_v1.ListOptions{})
	componentStatusList.Store(l)
	updateLastFetchedAt(key)
}

func getComponentStatusCompletions() []prompt.Suggest {
	go fetchComponentStatusList()
	l, ok := componentStatusList.Load().(*v1.ComponentStatusList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Config Maps */

var (
	configMapsList atomic.Value
)

func fetchConfigMapList(namespace string) {
	key := "config_map_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)
	l, _ := getClient().CoreV1().ConfigMaps(namespace).List(meta_v1.ListOptions{})
	configMapsList.Store(l)
}

func getConfigMapSuggestions() []prompt.Suggest {
	go fetchConfigMapList(v1.NamespaceAll)
	l, ok := configMapsList.Load().(*v1.ConfigMapList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Contexts */

var (
	contextList atomic.Value
)

func fetchContextList() {
	key := "context"
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)
	r, err := ExecuteAndGetResult("config get-contexts --no-headers -o name")
	if err != nil {
		log.Printf("[WARN] Got Error when fetchContextList: %s", err.Error())
	}
	contextList.Store(strings.Split(r, "\n"))
}

func getContextSuggestions() []prompt.Suggest {
	go fetchContextList()
	l, ok := contextList.Load().([]string)
	if !ok || len(l) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l))
	for i := range l {
		s[i] = prompt.Suggest{
			Text: l[i],
		}
	}
	return s
}

/* Pod */

var (
	podList *sync.Map
)

func fetchPods(namespace string) {
	key := "pod_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().Pods(namespace).List(meta_v1.ListOptions{})
	podList.Store(namespace, l)
}

func getPodSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchPods(namespace)
	x, ok := podList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.PodList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text:        l.Items[i].Name,
			Description: string(l.Items[i].Status.Phase),
		}
	}
	return s
}

func getPod(podName string) (v1.Pod, bool) {
	namespace := v1.NamespaceAll
	x, ok := podList.Load(namespace)
	if !ok {
		return v1.Pod{}, false
	}
	l, ok := x.(*v1.PodList)
	if !ok || len(l.Items) == 0 {
		return v1.Pod{}, false
	}
	for i := range l.Items {
		if podName == l.Items[i].Name {
			return l.Items[i], true
		}
	}
	return v1.Pod{}, false
}

func getPortsFromPodName(podName string) []prompt.Suggest {
	pod, found := getPod(podName)
	if !found {
		return []prompt.Suggest{}
	}

	// Extract unique ports
	portSet := make(map[int32]struct{})
	for i := range pod.Spec.Containers {
		ports := pod.Spec.Containers[i].Ports
		for j := range ports {
			portSet[ports[j].ContainerPort] = struct{}{}
		}
	}

	// Sort
	var ports []int
	for k := range portSet {
		ports = append(ports, int(k))
	}
	sort.Ints(ports)

	// Prepare suggestions
	suggests := make([]prompt.Suggest, 0, len(ports))
	for i := range ports {
		suggests = append(suggests, prompt.Suggest{
			Text: fmt.Sprintf("%d:%d", ports[i], ports[i]),
		})
	}
	return suggests
}

/* Daemon Sets */

var (
	daemonSetList *sync.Map
)

func fetchDaemonSetList(namespace string) {
	key := "daemon_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().AppsV1().DaemonSets(namespace).List(meta_v1.ListOptions{})
	daemonSetList.Store(namespace, l)
	return
}

func getDaemonSetSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchDaemonSetList(namespace)
	x, ok := daemonSetList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*apps_v1.DaemonSetList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Deployment */

var (
	deploymentList *sync.Map
)

func fetchDeployments(namespace string) {
	key := "deployment_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().AppsV1().Deployments(namespace).List(meta_v1.ListOptions{})
	deploymentList.Store(namespace, l)
	return
}

func getDeploymentSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchDeployments(namespace)
	x, ok := deploymentList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*apps_v1.DeploymentList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Endpoint */

var (
	endpointList *sync.Map
)

func fetchEndpoints(namespace string) {
	key := "endpoint_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().Endpoints(namespace).List(meta_v1.ListOptions{})
	endpointList.Store(key, l)
	return
}

func getEndpointsSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchEndpoints(namespace)
	x, ok := endpointList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.EndpointsList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Events */

var (
	eventList *sync.Map
)

func fetchEvents(namespace string) {
	key := "event_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().Events(namespace).List(meta_v1.ListOptions{})
	eventList.Store(namespace, l)
	return
}

func getEventsSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchEvents(namespace)
	x, ok := eventList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.EventList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Node */

var (
	nodeList atomic.Value
)

func fetchNodeList() {
	key := "node"
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().Nodes().List(meta_v1.ListOptions{})
	nodeList.Store(l)
	return
}

func getNodeSuggestions() []prompt.Suggest {
	go fetchNodeList()
	l, ok := nodeList.Load().(*v1.NodeList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Secret */

var (
	secretList *sync.Map
)

func fetchSecretList(namespace string) {
	key := "secret_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().Secrets(namespace).List(meta_v1.ListOptions{})
	secretList.Store(namespace, l)
	return
}

func getSecretSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchSecretList(namespace)
	x, ok := secretList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.SecretList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Ingress */

var (
	ingressList *sync.Map
)

func fetchIngressList(namespace string) {
	key := "ingress_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().ExtensionsV1beta1().Ingresses(namespace).List(meta_v1.ListOptions{})
	ingressList.Store(namespace, l)
	return
}

func getIngressSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchIngressList(namespace)

	x, ok := ingressList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.NamespaceList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* LimitRange */

var (
	limitRangeList *sync.Map
)

func fetchLimitRangeList(namespace string) {
	key := "limit_range_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().LimitRanges(namespace).List(meta_v1.ListOptions{})
	limitRangeList.Store(namespace, l)
	return
}

func getLimitRangeSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchLimitRangeList(namespace)
	x, ok := limitRangeList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.NamespaceList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* NameSpaces */

var (
	namespaceList atomic.Value
)

func fetchNameSpaceList() {
	key := "namespace"
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().Namespaces().List(meta_v1.ListOptions{})
	namespaceList.Store(l)
	return
}

func getNameSpaceSuggestions() []prompt.Suggest {
	go fetchNameSpaceList()
	l, ok := namespaceList.Load().(*v1.NamespaceList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Persistent Volume Claims */

var (
	persistentVolumeClaimsList *sync.Map
)

func fetchPersistentVolumeClaimsList(namespace string) {
	key := "persistent_volume_claims" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().PersistentVolumeClaims(namespace).List(meta_v1.ListOptions{})
	persistentVolumeClaimsList.Store(namespace, l)
	return
}

func getPersistentVolumeClaimSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchPersistentVolumeClaimsList(namespace)
	x, ok := persistentVolumeClaimsList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.PersistentVolumeClaimList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Persistent Volumes */

var (
	persistentVolumesList atomic.Value
)

func fetchPersistentVolumeList() {
	key := "persistent_volume"
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().PersistentVolumes().List(meta_v1.ListOptions{})
	persistentVolumesList.Store(l)
	return
}

func getPersistentVolumeSuggestions() []prompt.Suggest {
	go fetchPersistentVolumeList()
	l, ok := persistentVolumesList.Load().(*v1.PersistentVolumeList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Pod Security Policies */

var (
	podSecurityPolicyList atomic.Value
)

func fetchPodSecurityPolicyList() {
	key := "pod_security_policy"
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().ExtensionsV1beta1().PodSecurityPolicies().List(meta_v1.ListOptions{})
	podSecurityPolicyList.Store(l)
	return
}

func getPodSecurityPolicySuggestions() []prompt.Suggest {
	go fetchPodSecurityPolicyList()
	l, ok := podSecurityPolicyList.Load().(*ext_v1beta1.PodSecurityPolicyList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Pod Templates */

var (
	podTemplateList *sync.Map
)

func fetchPodTemplateList(namespace string) {
	key := "pod_template_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().PodTemplates(namespace).List(meta_v1.ListOptions{})
	podTemplateList.Store(namespace, l)
	return
}

func getPodTemplateSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchPodTemplateList(namespace)
	x, ok := podTemplateList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.PodTemplateList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Replica Sets */

var (
	replicaSetList *sync.Map
)

func fetchReplicaSetList(namespace string) {
	key := "replica_set_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().AppsV1().ReplicaSets(namespace).List(meta_v1.ListOptions{})
	replicaSetList.Store(namespace, l)
	return
}

func getReplicaSetSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchReplicaSetList(namespace)
	x, ok := replicaSetList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*apps_v1.ReplicaSetList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Replication Controller */

var (
	replicationControllerList *sync.Map
)

func fetchReplicationControllerList(namespace string) {
	key := "replication_controller" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().ReplicationControllers(namespace).List(meta_v1.ListOptions{})
	replicationControllerList.Store(namespace, l)
	return
}

func getReplicationControllerSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchReplicationControllerList(namespace)
	x, ok := replicationControllerList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.ReplicationControllerList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Resource quotas */

var (
	resourceQuotaList *sync.Map
)

func fetchResourceQuotaList(namespace string) {
	key := "resource_quota" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().ResourceQuotas(namespace).List(meta_v1.ListOptions{})
	resourceQuotaList.Store(namespace, l)
	return
}

func getResourceQuotasSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchResourceQuotaList(namespace)
	x, ok := resourceQuotaList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.ResourceQuotaList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Roles */

var (
	rolesList *sync.Map
)

func fetchRolesList(namespace string) {
	key := "roles" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().RbacV1().Roles(namespace).List(meta_v1.ListOptions{})
	resourceQuotaList.Store(namespace, l)
	return
}

func getRolesSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchRolesList(namespace)
	x, ok := rolesList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.ResourceQuotaList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}


/* Service Account */

var (
	serviceAccountList *sync.Map
)

func fetchServiceAccountList(namespace string) {
	key := "service_account_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().ServiceAccounts(namespace).List(meta_v1.ListOptions{})
	serviceAccountList.Store(namespace, l)
	return
}

func getServiceAccountSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchServiceAccountList(namespace)
	x, ok := serviceAccountList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.ServiceAccountList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}

/* Service */

var (
	serviceList *sync.Map
)

func fetchServiceList(namespace string) {
	key := "service_" + namespace
	if !shouldFetch(key) {
		return
	}
	updateLastFetchedAt(key)

	l, _ := getClient().CoreV1().Services(namespace).List(meta_v1.ListOptions{})
	serviceList.Store(namespace, l)
	return
}

func getServiceSuggestions() []prompt.Suggest {
	namespace := v1.NamespaceAll
	go fetchServiceList(namespace)
	x, ok := serviceList.Load(namespace)
	if !ok {
		return []prompt.Suggest{}
	}
	l, ok := x.(*v1.ServiceAccountList)
	if !ok || len(l.Items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(l.Items))
	for i := range l.Items {
		s[i] = prompt.Suggest{
			Text: l.Items[i].Name,
		}
	}
	return s
}
