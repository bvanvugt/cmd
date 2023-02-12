# cmd

```sh
go install github.com/bvanvugt/cmd@latest
```


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
