


To compile run 
```
GOOS=linux CGO_ENABLED=0 go build -ldflags '-w -s -extldflags "-static"' -o  cpuloader
```
