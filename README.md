# Admiral
![Builds Actions Status](https://github.com/phil-inc/admiral/workflows/Builds/badge.svg?branch=master)
![Releases Actions Status](https://github.com/phil-inc/admiral/workflows/Releases/badge.svg)
![Tags Actions Status](https://github.com/phil-inc/admiral/workflows/Tags/badge.svg)

Admiral is a lightweight service inspired by Kubewatch. It evolved out of a need
for extra observability of AWS EKS Fargate. Lack of control & vision of the
`kubelet` necessitates more consistent testing and reporting. 

Our ultimate goal is providing a simple interface and pattern for automating
routine Kubernetes operations.

## Available features

- Observe the following kinds of Kubernetes Events
    - Kill (scheduled pod death)
    - BackOff (unscheduled pod timeout)
    - NodeNotReady (unscheduled node failure)
    - Unhealthy (unscheduled pod failure)
- Send messages to a webhook
- Stream logs from pods to a logstore (currently supports Grafana Loki)
- Initiate performance testing on pod updates

## Application structure

Presently, the application depends on a single configuration file:
`${HOME}/.admiral.yaml`, which looks something like this:

```yaml
cluster: my-cluster
namespace: "" # Use all namespaces
events:
    handler:
        webhook:
            url: https://my.webhook.url
logstream:
    logstore:
        loki:
            url: https://loki.logging.svc.cluster.local:3100 # A svc named loki in the logging namespace
    apps: # The label "app" on a pod
        - my-app-deployment
ignorecontainers: [datadog-agent] # an array of container names to ignore
metrics:
  handler:
    prometheus: true
    pushgateway: "http://cluster.local:9091"
  apps:
    - my-app-name
```

Based on the config, the application instantiates a handler. For now, the only
available handler is webhook. It then instantiates a controller watching
the Kubernetes API server for a variety of defined events. Each controller adds
their events to a queue, which is then popped by the handler and POSTed to the
webhook.

## Building

Admiral is a statically compiled `golang` application and building is as simple
as:

```bash
CGO_ENABLED=0 GOOS=linux go build -o out/admiral ./cmd
```

You can also execute the above command with CMake:

```bash
make build
```

## Running

Admiral should be invoked with the single command `admiral`. It has 2 external
dependencies for success:

1. It needs access to a `kubeconfig`. Admiral will check the following
locations for a `kubeconfig`:
    1. In-cluster (native Kubernetes RBAC if it is a pod)
    2. `$HOME/.kube/config`
2. It needs a configuration file at `$HOME/.admiral.yaml`
