# Adding a config to Admiral

When you execute `admiral`, it expects a config file
at `$HOME/.admiral.yaml` (you can also specify a path
to a file with a flag). The config file gives details to
Admiral about how to run a given controller.

Each controller implements the generic controller interface
so it can `Watch` Kubernetes resources for changes and respond.
But, each controller has specific details that users will
customize through the config.

For example, in the `logs` controller, a user needs to specify
a backend where the `logs` end up streaming. The controller will
call a generic `Handle()` function, and the config will dictate
which backend is implemented. The backend will implement its own
`Handle()` function which will define how the log is processed.
If our backend is Loki, this means the log needs to be parsed
and formatted before an HTTP request ships it to Loki.

The config lets us define an interface where the user can
specify the fine detailed behavior of `admiral`. It lives in
[config/config.go][] & simply tries parsing the `yaml` file
into the structs defined in `config.go`.

[config/config.go]: ./config/config.go
