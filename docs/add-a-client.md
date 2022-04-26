# Adding a Client to Admiral

If you're adding a controller to Admiral, after you
[add a command][], you will need a client that can
handle the config and instantiate the appropriate
controller. This document will walk through setting
up a client to initialize the controller.

## Seting up the client

The client logic is not very sophisticated. It lives in
[pkg/client/client.go][]. On `Run()`, it checks for a valid
Kubernetes client credential, instantiates a Kubernetes
`InformerFactory`, then `switches` on the `operation`, where
it parses data from the `config` based on the actual command
issued.

So, for example, if you ran `admiral logs`, it would check
the environment for a Kubernetes client, it would instantiate
the `InformerFactory`, then it would Parse the config for
details related to the `logs` controller. These would be
details like the type of `Logstream`, the URL to the `Logstream`,
etc.

Basically, anything the user will pass to `admiral` via the config
file is addressed here before the controller is actually instantiated.

If the client validates the config and is able to instantiate the controller,
it will initialize the controller lifecycle, which will `Watch` Kubernetes
for specified resources. In the actual controller code, we will define
event handlers that respond to the watched Kubernetes activity.

[pkg/client/client.go]: ./pkg/client/client.go
