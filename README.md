# Lope

[![Build Status](https://travis-ci.org/Crazybus/lope.svg?branch=master)](https://travis-ci.org/Crazybus/lope)

Abuse docker as a development and testing environment that works (almost) identically across Linux/OSX/Windows. It creates a docker image with your current working directory and forwards environment variables/secrets/ssh agents so that you can run tooling and tests within a docker container with minimal boilerplate. 

# Why

It is 2018 and it is still hard to support running tooling across different operating systems in a consistent way. Having a single way to run pinned versions of tooling such as Ansible for local development and CI environments without needing to install and manage that particular tools equivalent of python virtualenvs. 

Lope is an attempt to make a more structured version of this bash one liner with support for more advanced patterns like ssh forwarding:
```
docker run --entrypoint=/bin/sh $(env | grep ^AWS | awk -F= '{ print "--env " $1; }' | tr "\n" ' ') --rm -i -v $(pwd):/app -w /app --user=$(id -u):$(id -g) hashicorp/terraform:${TERRAFORM_VERSION} -c "terraform plan"
```
With lope:
```
lope hashicorp/terraform:${TERRAFORM_VERSION} terraform plan
```

With `.lope.yml` file:
```
image: hashicorp/terraform:${TERRAFORM_VERSION}
command: terraform plan
env:
  - ^AWS
``` 

## Design goals

* Docker is the only dependency needed for developing/testing/deploying
* Defaults which favor usability over speed and security while still having configuration options available to restrict which environment variables/secrets are forwarded into containers for proper usage. 
* All actions are immutable. All files and dependencies are added to a docker image before running. This means no local state is modified and the state inside the container is static during its lifetime. This makes it possible to run something like `ansible-playbook` and then immediately switch to a different branch and continue to make changes.

## Examples

Usage: `lope image commands go here`

```
$ lope alpine 'ls'
README.md
lope
lope.go
```
```
$ lope alpine 'cat /etc/issue'
Welcome to Alpine Linux 3.7 Kernel \r on an \m (\l)
```
```
$ lope vault vault read secret/test/hellope
Key                 Value
---                 -----
refresh_interval    768h
value               world
```

```
$ lope lachlanevenson/k8s-kubectl kubectl get pods
NAME                    READY     STATUS    RESTARTS   AGE
nginx-7c87f569d-5zvx4   1/1       Running   0          13s
```

## Features

### Planned

* Add yaml file to define configuration instead of doing a big one liner
* Add option in yaml file to specify mounted files
* Add yaml file option to include/exclude environment variables with pattern support
* Automated ssh agent forwarding for OSX. https://github.com/avsm/docker-ssh-agent-forward
* Add simple build step options. E.g. alpine as base image with a single `RUN apk add package`. 
* Add option to use bind mounts for adding the current working directory. Also allow disabling mounting current directory altogether for use cases like `vault status`
* Add default .dockerignore for things like .git and .vagrant
* Automatically expose ports from Dockerfile

### Done

* Run a docker container with current directory added
* Forward all environment variables into the container
* Add blacklist to skip certain environment variables
* Mount secret files and directories into the container (like ~/.vault-token)
* Automatically add well known locations for secrets for development usage
