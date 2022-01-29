# Requirements

## Start-up

- It should return an error if there's too many arguments
    - logs
    - events
    - main
- It should return an error if there are invalid arguments
    - logs
    - events
    - main
- It should initialize its controller
    - logs
    - events
- It should return a helpful message
    - main
- It should load a config from (first = more precedence):
    1. `--config-file` or `-f`
    2. `$HOME/.admiral.yaml`
    3. ENV

## Config

- It should return an error for any missing values
- It should return an implementation of a backend interface
    - logStore
    - eventHandler
- It should return a valid `kubeconfig`

## Events

## Logs
- It should 