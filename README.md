# Project Resource Quota
A Kubernetes CRD + Controller to set resource quotas across multiple namespaces.

## Description
Kubernetes has a built-in resource quota mechanism to limit specific resources (CPU, memory, storage, etc.) per namespace.
This project extends the native Kubernetes built-in resource quota mechanism, making the resource quotas across multiple namespaces (project).

The design concepts are:
1. Create a CRD `projectresourcequotas.jenting.io` to define the per-project resource quotas.
1. The user creates the `projectresourcequotas.jenting.io` CRs with namespaces + resource quotas limits.
1. Have a controller to calculate current resource usage and updates the current resource usage to `projectresourcequotas.jenting.io` CRs status.
1. Have admission webhooks for rejecting the Kubernetes resources creation/modification if `current resource usage + request resource limit > project resource quota limit`. The supported Kubernetes resources are:
   - ConfigMap
   - PersistentVolumeClaim
   - Pod
   - ReplicationController
   - ResourceQuota
   - Secret
   - Service
1. Have an admission webhook for rejecting:
   - the ProjectResourceQuota CR creation if the namespace is configured in another ProjectResourceQuota CR already.
   - the ProjectResourceQuota CR modification if the `current resource usage > updated project resource quota limit`.

The `projectresourcequotas.jenting.io` CR supports resource quotas are:
| Resource Name | Description |
| :-----| :----- |
| configmaps | The total number of ConfigMaps within the project cannot exceed this value. |
| persistentvolumeclaims | The total number of PersistentVolumeClaims within the project cannot exceed this value. |
| pods | Across all pods in a non-terminal state (.status.Phase != (Failed, Succeeded)) within the project, the total number of Pods cannot exceed this value. |
| requests.cpu | Across all pods in a non-terminal state (.status.Phase != (Failed, Succeeded)) within the project, the sum of CPU requests cannot exceed this value. Note that, it requires that every incoming container makes explicit `requests.cpu`. |
| requests.memory | Across all pods in a non-terminal state within the project, the sum of memory requests cannot exceed this value. It requires that every incoming container makes explicit `requests.memory`. |
| requests.storage | Across all pods in the project, the sum of local ephemeral storage requests cannot exceed this value. |
| requests.ephemeral-storage | Across all pods in the project, the sum of local ephemeral storage requests cannot exceed this value. |
| cpu | Same as `requests.cpu`. |
| memory | Same as `requests.memory`. |
| storage | Same as `requests.storage`. |
| ephemeral-storage | Same as `requests.ephemeral-storage`. |
| limits.cpu | Across all pods in a non-terminal state with the project, the sum of CPU limits cannot exceed this value. It requires that every incoming container makes explicit `limit.cpu`. |
| limits.memory | Across all pods in a non-terminal state within the project, the sum of memory limits cannot exceed this value. It requires that every incoming container makes explicit `limit.memory`. |
| limits.ephemeral-storage | Across all pods in the project, the sum of local ephemeral storage limits cannot exceed this value. |
| replicationcontrollers | The total number of ReplicationControllers within the project cannot exceed this value. |
| resourcequotas | The total number of ResourceQuotas within the project cannot exceed this value. |
| secrets | The total number of Secrets within the project cannot exceed this value. |
| services | The total number of Services within the project cannot exceed this value. |
| services.loadbalancers | The total number of Services of type LoadBalancer within the project cannot exceed this value. |
| services.nodeports | The total number of Services of type NodePort within the project cannot exceed this value. |

> **Note**
> All the supported resource quotas are per-namespace.

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Prerequisite
1. Install cert-manager:
   ```sh
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.11.0/cert-manager.yaml
   ```

2. Install the CRDs into the cluster:
   ```sh
   make install
   ```

### Deployment
1. Build and push your image to the location specified by `IMG`:
   ```sh
   make docker-build docker-push IMG=<some-registry>/resourcequota:tag
   ```

2. Deploy the controller to the cluster with the image specified by `IMG`:
   ```sh
   make deploy IMG=<some-registry>/resourcequota:tag
   ```

### Usage
1. View the `projectresourcequotas.jenting.io` CRs:
   ```sh
   kubectl get prq
   ```

2. Install the CRs:
   ```sh
   kubectl apply -f config/samples/_v1_projectresourcequota.yaml
   ```

3. Install another CR. Verify the installation fails because the namespace is configured in another CR already:
   ```sh
   kubectl apply -f config/samples/_v2_projectresourcequota.yaml
   ```

4. Install the Pods. Verify the second pod installation fails because the resource request + used > hard limit:
   ```sh
   kubectl apply -f config/samples/pod-nginx.yaml
   kubectl apply -f config/samples/pod-busybox.yaml
   ```

5. Remove the `default` from the `projectresourcequotas.jenting.io` CR. Verify the ``projectresourcequotas.jenting.io`` CR `status.used` reflecting the current status.
   ```sh
   # get current projectresourcequota-sample 
   kubectl get prq projectresourcequota-sample

   # remove the default namespace from spec.namespaces
   kubectl edit prq projectresourcequota-sample

   # check the projectresourcequota-sample is updated
   kubectl get prq projectresourcequota-sample
   ```

> **Note**
> We don't support calculating the existing Kubernetes resources usage before the ProjectResourceQuota CR is configured. It means for the existing Kubernetes resources are not limited by the new ProjectResourceQuota CR.

### Uninstall
1. Undeploy the resources from the cluster:
   ```sh
   make undeploy
   ```

2. Uninstall the CRDs from the cluster:
   ```sh
   make uninstall
   ```

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.
