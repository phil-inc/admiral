# Adding a controller to Admiral

Once you have a command, a config, and
the client running, the controller defines
the actual behavior of `admiral`.

Every controller lives in [pkg/controllers/][].
Each controller must implement the generic interface
in `controller.go`, which just requires a `Watch()` function.

The controller `Watch()` & `Run()` functions are boilerplate
for setting up the listener listening to Kubernetes.

When instantiating a new controller (some function like
`NewCommandController()`), we must instantiate an `Informer`
which will hold `EventHandlers` from Kubernetes. These define
the response of the controller when it observes certain Kubernetes
behaviors.

These should be functions leading us into the logic of what we
actually want the command to accomplish. For example in `logs`,
we handle every pod creation, update, and delete. On creations,
we check if the pod is listed in our config & if its status is running.
Then, we execute the functions that will stream logs to our backend.
On updates, we check to see if the pod is already being streamed, and
if it is not, we stream logs to our backend.
On deletions, we stop streaming logs to our backend.

[pkg/controllers/]: ./pkg/controllers
