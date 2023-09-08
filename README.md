# Admiral

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

`make` does the following:

```bash
make tidy  # go mod tidy
make fmt   # go fmt
make test  # go test
make build # go build

make       # all of the above
```

## Running

Admiral should be invoked with the single command `admiral`. It has 2 external
dependencies for success:

1. It needs access to a `kubeconfig`. Admiral will check the following
locations for a `kubeconfig`:
    1. In-cluster (native Kubernetes RBAC if it is a pod)
    2. `$HOME/.kube/config`
2. It needs a configuration file at `$HOME/.admiral.yaml`
