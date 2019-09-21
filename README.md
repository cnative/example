# Example Service
Example Micro Service with GO


Pre-Req

- [Go 1.3+]
- [Docker]
- [Direnv]
- Editor ( ex: [VS Code])

To Install the pre-reqs, which is usually a one time setup of your dev env


```bash
brew install go direnv
brew cask install visual-studio-code
```

## Build

to build the source

```bash
make clean build
```

`NOTE:` This will download all the required tool dependencies for the first time and add them to your `<PROJECT_ROOT>/.tools` folder.  You can run `make install-deptools` to only install dependent tools. See [Makefile] for additional targets

## Local kubernetes dev cluster

to bring up a local kubernetes dev cluser with postgres 

```
make cluster-local
```

`NOTE:` In order for the above command to work please make sure [docker] is installed. The resulting cluster is based on [Kubernetes IN Docker] (aka _kind_)

If already have run `make clean build` then the tools folder would have [kubectl] which you could use to talk to the kubernetes cluster.

To use the local cluster you need to set the KUBECONFIG

```bash
export KUBECONFIG=$(kind get kubeconfig-path --name cnative-local)
```

```bash
kubectl get pods -A
```

or you can create an alias like 

```bash
alias lk='kubectl --kubeconfig=$(kind get kubeconfig-path --name cnative-local)'
```

```bash
lk get pods -A
```

to clean up the local cluster

```bash
make cluster-local-delete
```

<!--links-->
[go 1.3+]: https://golang.org/
[docker]: https://hub.docker.com/editions/community/docker-ce-desktop-mac
[Direnv]: https://direnv.net
[VS Code]: https://code.visualstudio.com/Download
[Kubernetes IN Docker]: https://github.com/kubernetes-sigs/kind
[kubectl]: https://kubernetes.io/docs/reference/kubectl/kubectl/
[Makefile]: ./Makefile