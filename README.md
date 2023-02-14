# cmd

```sh
go install github.com/bvanvugt/cmd@latest
```

Example Config:
```yaml
# .devcontainer/cmd.yaml

devcontainer:
  name: cmd.local
env:
  EXAMPLE: value
commands:
  run:
    shell: while true; do echo 'barf?'; sleep 1; done
```

Dev Config:
```yaml
devcontainer:
  name: cmd.local
  dir: /workspaces/cmd
env:
  EXAMPLE: value
commands:
  install:
    shell: go install .
  run:
    shell: go run cmd.go barf
  barf:
    shell: while true; do echo 'barf?'; sleep 1; done
  test:
    shell: echo test!!!
  pwd:
    shell: pwd
```
