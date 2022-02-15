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

## Config

- It should return an error for any missing values
- It should return an implementation of a backend interface
    - logStore
    - eventHandler
- It should return a valid `kubeconfig`
- It should load a config from (first = more precedence):
    1. `--config-file` or `-f`
    2. `$HOME/.admiral.yaml`

## Events

## Logs
- It should watch all of the pods
    - OnAdd()
        - It should `continue` if the pod is NOT running
        - It should check if the pod is already handled
            - It should handle the pod or do nothing
    - OnChange()
        - It should check if the pod is running
            - It should check if the pod is already handled
                - It should handle the pod or do nothing
        - It should check if the pod is succeeded or failed
            - It should check if the pod is already finished
                - It should finish the pod or do nothing
    - OnDelete()
        - It should check if the pod is deleted
            - It should delete the pod or do nothing