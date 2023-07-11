# Admiral
![Builds Actions Status](https://github.com/phil-inc/admiral/workflows/Builds/badge.svg?branch=master)
![Releases Actions Status](https://github.com/phil-inc/admiral/workflows/Releases/badge.svg)
![Tags Actions Status](https://github.com/phil-inc/admiral/workflows/Tags/badge.svg)

Admiral is an in-memory service evolved out of a need
for extra observability of AWS EKS Fargate. Lack of control & vision of the
`kubelet` necessitates more consistent testing and reporting. 

Our ultimate goal is providing a simple interface and pattern for automating
routine Kubernetes operations.

## Available features

- Observe all kinds of Kubernetes Events
- Send messages to a webhook
- Stream logs from pods to a logstore (currently supports Grafana Loki)

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
