/*
Copyright 2023 The Kubernetes Authors.

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

package helper

import (
        "context"
        "encoding/json"
        "fmt"
        "net"
        "os"
        "os/exec"
        "testing"
        "time"

        appsv1 "k8s.io/api/apps/v1"
        "k8s.io/utils/pointer"

        corev1 "k8s.io/api/core/v1"
        apierrors "k8s.io/apimachinery/pkg/api/errors"
        "k8s.io/apimachinery/pkg/api/meta"
        metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        "k8s.io/apimachinery/pkg/types"
        "sigs.k8s.io/e2e-framework/klient/k8s"
        "sigs.k8s.io/e2e-framework/klient/k8s/resources"
        "sigs.k8s.io/e2e-framework/klient/wait"
        "sigs.k8s.io/e2e-framework/klient/wait/conditions"
        "sigs.k8s.io/e2e-framework/pkg/env"
        "sigs.k8s.io/e2e-framework/pkg/envconf"
        "sigs.k8s.io/e2e-framework/pkg/features"

        "sigs.k8s.io/kwok/pkg/log"
        "sigs.k8s.io/kwok/pkg/utils/slices"
)

// nodeIsReady returns a function that checks if a node is ready
func nodeIsReady(name string) func(obj k8s.Object) bool {
        return func(obj k8s.Object) bool {
                node := obj.(*corev1.Node)
                if node.Name != name {
                        return false
                }
                cond, ok := slices.Find(node.Status.Conditions, func(cond corev1.NodeCondition) bool {
                        return cond.Type == corev1.NodeReady
                })
                if ok && cond.Status == corev1.ConditionTrue {
                        return true
                }
                return false
        }
}

// CreateNode creates a node and waits for it to be ready
func CreateNode(node *corev1.Node) features.Func {
        return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
                client, err := resources.New(c.Client().RESTConfig())
                if err != nil {
                        t.Fatal(err)
                }

                t.Log("creating node", node.Name)
                err = client.Create(ctx, node)
                if err != nil {
                        t.Fatal(err)
                }
                t.Log("waiting for node to be ready", node.Name)
                err = wait.For(
                        conditions.New(client).ResourceMatch(node, nodeIsReady(node.Name)),
                        wait.WithContext(ctx),
                        wait.WithTimeout(20*time.Minute),
                )
                if err != nil {
                        t.Fatal(err)
                }
                t.Log("node is ready", node.Name)
                return ctx
        }
}

// DeleteNode deletes a node
func DeleteNode(node *corev1.Node) features.Func {
        return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
                client, err := resources.New(c.Client().RESTConfig())
                if err != nil {
                        t.Fatal(err)
                }

                t.Log("deleting node", node.Name)
                err = client.Delete(ctx, node)
                if err != nil {
                        t.Fatal(err)
                }

                err = wait.For(
                        conditions.New(client).ResourceDeleted(node),
                        wait.WithContext(ctx),
                        wait.WithTimeout(20*time.Minute),
                )
                if err != nil {
                        t.Fatal(err)
                }
                return ctx
        }
}

// CreatePod creates a pod and waits for it to be ready
func CreatePod(pod *corev1.Pod) features.Func {
        return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
                client, err := resources.New(c.Client().RESTConfig())
                if err != nil {
                        t.Fatal(err)
                }

                t.Log("creating pod", log.KObj(pod))
                err = client.Create(ctx, pod)
                if err != nil {
                        t.Fatal(err)
                }

                t.Log("waiting for pod to be ready", log.KObj(pod))
                err = wait.For(
                        conditions.New(client).PodConditionMatch(pod, corev1.PodReady, corev1.ConditionTrue),
                        wait.WithContext(ctx),
                        wait.WithTimeout(20*time.Minute),
                )
                if err != nil {
                        t.Fatal(err)
                }

                err = client.Get(ctx, pod.GetName(), pod.GetNamespace(), pod)
                if err != nil {
                        t.Fatal(err)
                }

                if pod.Spec.NodeName == "" {
                        t.Fatal("pod node name is empty", log.KObj(pod))
                }

                if pod.Status.PodIP != "" {
                        if pod.Spec.HostNetwork {
                                if pod.Status.PodIP != pod.Status.HostIP {
                                        t.Errorf("pod ip %q is not equal to host ip %q: %s", pod.Status.PodIP, pod.Status.HostIP, log.KObj(pod))
                                }
                        } else {
                                if pod.Status.PodIP == pod.Status.HostIP {
                                        t.Errorf("pod ip %q is equal to host ip %q: %s", pod.Status.PodIP, pod.Status.HostIP, log.KObj(pod))
                                }

                                var node corev1.Node
                                err = client.Get(ctx, pod.Spec.NodeName, "", &node)
                                if err != nil {
                                        t.Fatal(err)
                                }

                                if node.Spec.PodCIDR != "" {
                                        _, ipnet, err := net.ParseCIDR(node.Spec.PodCIDR)
                                        if err != nil {
                                                t.Errorf("failed to parse pod cidr %q in %q", node.Spec.PodCIDR, node.Name)
                                        }

                                        ip := net.ParseIP(pod.Status.PodIP)
                                        if !ipnet.Contains(ip) {
                                                t.Errorf("pod ip %q is not in pod cidr %q in %q: %s", pod.Status.PodIP, node.Spec.PodCIDR, node.Name, log.KObj(pod))
                                        }
                                }
                        }
                }

                t.Log("pod is ready", log.KObj(pod))
                return ctx
        }
}

// DeletePod deletes a pod
func DeletePod(pod *corev1.Pod) features.Func {
        return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
                client, err := resources.New(c.Client().RESTConfig())
                if err != nil {
                        t.Fatal(err)
                }

                t.Log("deleting pod", log.KObj(pod))
                err = client.Delete(ctx, pod)
                if err != nil {
                        t.Fatal(err)
                }

                err = wait.For(
                        conditions.New(client).ResourceDeleted(pod),
                        wait.WithContext(ctx),
                        wait.WithTimeout(20*time.Minute),
                )
                if err != nil {
                        t.Fatal(err)
                }
                return ctx
        }
}

// WaitForAllNodesReady waits for all nodes to be ready
func WaitForAllNodesReady() env.Func {
        return func(ctx context.Context, c *envconf.Config) (context.Context, error) {
                client, err := resources.New(c.Client().RESTConfig())
                if err != nil {
                        return ctx, err
                }

                logger := log.FromContext(ctx)

                var list corev1.NodeList
                err = wait.For(
                        func(ctx context.Context) (done bool, err error) {
                                if err = client.List(ctx, &list); err != nil {
                                        logger.Error("failed to list nodes", err)
                                        return false, nil
                                }

                                metaList, err := meta.ExtractList(&list)
                                if err != nil {
                                        logger.Error("failed to extract list", err)
                                        return false, nil
                                }
                                if len(metaList) == 0 {
                                        logger.Error("no node found", nil)
                                        return false, nil
                                }

                                notReady := []string{}
                                for _, obj := range metaList {
                                        node := obj.(*corev1.Node)
                                        cond, ok := slices.Find(node.Status.Conditions, func(cond corev1.NodeCondition) bool {
                                                return cond.Type == corev1.NodeReady
                                        })
                                        if !ok || cond.Status != corev1.ConditionTrue {
                                                notReady = append(notReady, node.Name)
                                        }
                                }
                                if len(notReady) != 0 {
                                        logger.Error("not ready nodes", fmt.Errorf("%v", notReady))
                                        return false, nil
                                }

                                return true, nil
                        },
                        wait.WithContext(ctx),
                        wait.WithTimeout(20*time.Minute),
                )
                if err != nil {
                        return ctx, err
                }

                return ctx, nil
        }
}

//

// WaitForAllNodesReady waits for all nodes to be ready
func getLeaseTransitions(want int, nodeName string, namespace string) error {
        // Execute the kubectl command
        cmd := exec.Command("kubectl", "get", "leases", nodeName, "-n", namespace, "-ojson")
        out, err := cmd.Output()
        if err != nil {
                return fmt.Errorf("failed to execute kubectl comd: %w", err)
        }

        // Define a struct to unmarshal the JSON output
        var result struct {
                Spec struct {
                        LeaseTransitions int `json:"leaseTransitions"`
                } `json:"spec"`
        }

        // Unmarshal the JSON output
        if err := json.Unmarshal(out, &result); err != nil {
                return fmt.Errorf("failed to unmarshal JSON output: %w", err)
        }
        if result.Spec.LeaseTransitions != want {
                return fmt.Errorf("lease transitions  want %d", want)
        }

        return nil
}

// Function to get lease transitions

// WaitForAllPodsReady waits for all pods to be ready
func ChangePods() env.Func {
        return func(ctx context.Context, c *envconf.Config) (context.Context, error) {
                client, err := resources.New(c.Client().RESTConfig())
                if err != nil {
                        return ctx, err
                }

                logger := log.FromContext(ctx)

                var list corev1.PodList
                err = wait.For(
                        func(ctx context.Context) (done bool, err error) {
                                if err = client.List(ctx, &list); err != nil {
                                        logger.Error("failed to list pods", err)
                                        return false, nil
                                }

                                metaList, err := meta.ExtractList(&list)
                                if err != nil {
                                        logger.Error("failed to extract list", err)
                                        return false, nil
                                }
                                if len(metaList) == 0 {
                                        logger.Error("no pod found", nil)
                                        return false, nil
                                }

                                for _, obj := range metaList {
                                        pod := obj.(*corev1.Pod)
                                        // On Kind, ignore pods in kube-system and local-path-storage namespaces
                                        if pod.Namespace == "kube-system" || pod.Namespace == "local-path-storage" {
                                                continue
                                        }else{ 
                                        mergePatch, err := json.Marshal(map[string]interface{}{
                                                "metadata": map[string]interface{}{
                                                        "annotations": map[string]interface{}{
                                                                "kwok.x-k8s.io/status":"custom",
                                                        },
                                                },
                                        })
                                        if err != nil {
                                                logger.Error("error while json marshalling", err)
                                        }

                                        err = client.Patch(context.Background(), pod, k8s.Patch{PatchType: types.StrategicMergePatchType, Data: mergePatch})
                                        if err != nil {
                                                logger.Error("error while patching the node", err)
                                        }
                                mergePatch, err = json.Marshal(map[string]interface{}{
                                                "status": map[string]interface{}{
                                                        "podIP": "192.168.0.1",
                                                },
                                        })
                                        if err != nil {
                                                logger.Error("error while json marshalling", err)
                                        }

                                        err = client.Patch(context.Background(), pod, k8s.Patch{PatchType: types.StrategicMergePatchType, Data: mergePatch})
                                        if err != nil {
                                                logger.Error("error while patching the pod", err)
                                        }
                                        if pod.Status.PodIP == "192.168.0.1" {
                                                logger.Error("Error: pod is not updated", nil)

                                        }
                                        break
                                }
                                }

                                return true, nil
                        },
                        wait.WithContext(ctx),
                        wait.WithTimeout(20*time.Minute),
                )
                if err != nil {
                        return ctx, err
                }
 
                return ctx, nil
        }
}

// WaitForAllPodsReady waits for all pods to be ready
func ChangePs() env.Func {
        return func(ctx context.Context, c *envconf.Config) (context.Context, error) {
                client, err := resources.New(c.Client().RESTConfig())
                if err != nil {
                        return ctx, err
                }

                logger := log.FromContext(ctx)

                var list corev1.NodeList
                err = wait.For(
                        func(ctx context.Context) (done bool, err error) {
                                if err = client.List(ctx, &list); err != nil {
                                        logger.Error("failed to list nodes", err)
                                        return false, nil
                                }

                                metaList, err := meta.ExtractList(&list)
                                if err != nil {
                                        logger.Error("failed to extract list", err)
                                        return false, nil
                                }
                                if len(metaList) == 0 {
                                        logger.Error("no node found", nil)
                                        return false, nil
                                }

                                for _, obj := range metaList {
                                        node := obj.(*corev1.Node)
                                        if node.ObjectMeta.Name == "fake-node" {

                                                mergePatch, err := json.Marshal(map[string]interface{}{
                                                        "metadata": map[string]interface{}{
                                                                "annotations": map[string]interface{}{
                                                                        "kwok.x-k8s.io/status":"custom",
                                                                },
                                                        },
                                                })
                                                if err != nil {
                                                        logger.Error("error while json marshalling", err)
                                                }

                                                err = client.Patch(context.Background(), node, k8s.Patch{PatchType: types.StrategicMergePatchType, Data: mergePatch})
                                                if err != nil {
                                                        logger.Error("error while patching the node", err)
                                                }

                                                mergePatch, err = json.Marshal(map[string]interface{}{
                                                        "status": map[string]interface{}{
                                                                "nodeInfo": map[string]interface{}{
                                                                        "kubeletVersion": "fake-new",
                                                                },
                                                        },
                                                })
                                                if err != nil {
                                                        logger.Error("error while json marshalling", err)
                                                }

                                                err = client.Patch(context.Background(), node, k8s.Patch{PatchType: types.StrategicMergePatchType, Data: mergePatch})
                                                if err != nil {
                                                        logger.Error("error while patching the node", err)
                                                }
                                                if node.Status.NodeInfo.KubeletVersion == "fake-node" {
                                                        logger.Error("Error: fake-node is not updated", nil)

                                                }
                                        }
                                        fmt.Printf(node.GetAnnotations()["kwok.x-k8s.io/status"])

                                }
                                return true, nil
                        },
                        wait.WithContext(ctx),
                        wait.WithTimeout(20*time.Minute),
                )
                if err != nil {
                        return ctx, err
                }

                return ctx, nil
        }
}

// WaitForAllPodsReady waits for all pods to be ready
func WaitForAllPodsReady() env.Func {
        return func(ctx context.Context, c *envconf.Config) (context.Context, error) {
                client, err := resources.New(c.Client().RESTConfig())
                if err != nil {
                        return ctx, err
                }

                logger := log.FromContext(ctx)
                var list corev1.PodList
                err = wait.For(
                        func(ctx context.Context) (done bool, err error) {
                                if err = client.List(ctx, &list); err != nil {
                                        logger.Error("failed to list pods", err)
                                        return false, nil
                                }

                                metaList, err := meta.ExtractList(&list)
                                if err != nil {
                                        logger.Error("failed to extract list", err)
                                        return false, nil
                                }
                                if len(metaList) == 0 {
                                        logger.Error("no pod found", nil)
                                        return false, nil
                                }

                                notReady := []string{}
                                for _, obj := range metaList {
                                        pod := obj.(*corev1.Pod)
                                        // On Kind, ignore pods in kube-system and local-path-storage namespaces
                                        if pod.Namespace == "kube-system" || pod.Namespace == "local-path-storage" {
                                                continue
                                        }
                                        if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded {
                                                notReady = append(notReady, log.KObj(pod).String())
                                        }
                                }
                                if len(notReady) != 0 {
                                        logger.Error("not ready pods", fmt.Errorf("%v", notReady))
                                        return false, nil
                                }

                                return true, nil
                        },
                        wait.WithContext(ctx),
                        wait.WithTimeout(20*time.Minute),
                )
                if err != nil {
                        return ctx, err
                }

                return ctx, nil
        }
}

func waitForServiceAccountReady(ctx context.Context, resource *resources.Resources, name, namespace string) error {
        var sa corev1.ServiceAccount

        logger := log.FromContext(ctx)

        err := wait.For(
                func(ctx context.Context) (done bool, err error) {
                        err = resource.Get(ctx, name, namespace, &sa)
                        if err == nil {
                                return true, nil
                        }
                        if !apierrors.IsNotFound(err) {
                                logger.Error("failed to get service account", err)
                                return false, nil
                        }

                        err = resource.Create(ctx, &corev1.ServiceAccount{
                                ObjectMeta: metav1.ObjectMeta{
                                        Name:      name,
                                        Namespace: namespace,
                                },
                        })
                        if err == nil {
                                return false, nil
                        }
                        if apierrors.IsAlreadyExists(err) {
                                return false, nil
                        }

                        logger.Error("failed to create service account", err)
                        return false, nil
                },
                wait.WithContext(ctx),
                wait.WithTimeout(10*time.Minute),
        )
        if err != nil {
                return fmt.Errorf("wait for %s.%s service account ready: %w", name, namespace, err)
        }
        return nil
}

// Environment returns an environment of the test
func Environment() env.Environment {
        logger := log.NewLogger(os.Stderr, log.LevelDebug)
        cfg, err := envconf.NewFromFlags()
        if err != nil {
                logger.Error("failed to create config", err)
                os.Exit(1)
        }

        ctx := context.Background()
        ctx = log.NewContext(ctx, logger)

        testEnv, err := env.NewWithContext(ctx, cfg)
        if err != nil {
                logger.Error("failed to create environment", err)
                os.Exit(1)
        }

        return testEnv
}

//recreatekwok re-create kwok


func RecreateKwok() env.Func {
        return func(ctx context.Context, c *envconf.Config) (context.Context, error) {
                err := getLeaseTransitions(0, "fake-node", "kube-node-lease")
                if err != nil {
                        return ctx, err
                }
                logger := log.FromContext(ctx)

                res, err := resources.New(c.Client().RESTConfig())
                if err != nil {
                        logger.Error("Error creating new resources object: %v", err)
                        return ctx, err
                }

                deps := &appsv1.DeploymentList{}
                err = res.List(context.TODO(), deps)
                if err != nil {
                        logger.Error("error while getting the deployment", err)
                        return ctx, err
                }

                if deps.Items == nil {
                        logger.Error("error while getting the list of deployments", err)
                        return ctx, err
                }
                //scaleing replica
                for _, dep := range deps.Items {
                        if dep.Name == "kwok-controller" {
                                if dep.Namespace == "kube-system" {
                                        dep.Spec.Replicas = pointer.Int32(0)
                                        err = res.Update(context.TODO(), &dep)
                                        if err != nil {
                                                logger.Error("error while updating deployment", err)
                                                return ctx, err
                                        }
                                }
                        }
                }

                //Delete pods in kuwe system
                pods := &corev1.PodList{}
                err = wait.For(
                        func(ctx context.Context) (done bool, err error) {
                                err = res.List(context.TODO(), pods)
                                if err != nil {
                                        logger.Error("error while getting the list of pods", err)
                                        return false, err

                                }

                                if pods.Items == nil {
                                        logger.Error("error while getting the list of pods", err)
                                        return false, err

                                }

                                for _, pod := range pods.Items {
                                        if pod.Namespace == "kube-system" {
                                                if pod.ObjectMeta.Labels["app"] == "kwok-controller" {
                                                        err = res.Delete(context.TODO(), &pod)
                                                        if err != nil {
                                                                logger.Error("error while deleting pod", err)
                                                                return false, err
                                                        }
                                                }
                                        }
                                }
                                return true, nil
                        },
                        wait.WithContext(ctx),
                        wait.WithTimeout(20*time.Minute),
                )
                //rescale the replica
                for _, dep := range deps.Items {
                        if dep.Name == "deployment/kwok-controller" {
                                if dep.Namespace == "kube-system" {
                                        dep.Spec.Replicas = pointer.Int32(5)
                                        err = res.Update(context.TODO(), &dep)
                                        if err != nil {
                                                logger.Error("error while updating deployment", err)
                                                return ctx, err
                                        }
                                }
                        }
                }
                if err != nil {
                        return ctx, err
                }
                //delete all pods in defau;t

                err = wait.For(
                        func(ctx context.Context) (done bool, err error) {
                                err = res.List(context.TODO(), pods)
                                if err != nil {
                                        logger.Error("error while getting the list of pods", err)
                                        return false, err

                                }

                                if pods.Items == nil {
                                        logger.Error("error while getting the list of pods", err)
                                        return false, err

                                }
                                //delete all pods in defau;t
                                for _, pod := range pods.Items {
                                        if pod.Namespace == "default" {
                                                err = res.Delete(context.TODO(), &pod)
                                                if err != nil {
                                                        logger.Error("error while deleting pod", err)
                                                        return false, err
                                                }
                                        }
                                }
                                return true, nil
                        },
                        wait.WithContext(ctx),
                        wait.WithTimeout(20*time.Minute),
                )

                err = getLeaseTransitions(1, "fake-node", "kube-node-lease")
                if err != nil {
                        return ctx, err
                }
                return ctx, nil
        }
	}