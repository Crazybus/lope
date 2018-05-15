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

With `.lope.yml` file (not yet implemented):
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
* Testing and tooling should be run the same in development and CI environments

## Command line flags
| Flag                   | Effect                                                                                                                                           |
|------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------|
| -addMount              | Setting this will add the directory into the image instead of mounting it                                                                        |
| -blacklist _string_    | Comma seperated list of environment variables that will be ignored by lope (default `"HOME,SSH_AUTH_SOCK"`)                                      |
| -dir _string_          | The directory that will be mounted into the container. Defaut is current working directory                                                       |
| -docker                | Mount the docker socket inside the container (default `true`)                                                                                    |
| -dockerSocket _string_ | Path to the docker socket (default `"/var/run/docker.sock"`)                                                                                     |
| -entrypoint _string_   | The entrypoint for running the lope command (default `/bin/sh`)                                                                                  |
| -instruction _value_   | Extra docker image instructions to run when building the image. Can be specified multiple times                                                  |
| -noMount               | Disable mounting the current working directory into the image                                                                                    |
| -noSSH                 | Disable forwarding ssh agent into the container                                                                                                  |
| -path _value_          | Paths that will be mounted from the users home directory into lope. Path will be ignored if it isn't accessible. Can be specified multiple times |
| -whitelist _string_    | Comma seperated list of environment variables that will be be included by lope                                                                   |


## Examples

Usage: `lope [<flags>] <docker image> <commands go here>`

Run `lope -help` for all command line flags.

```
$ lope alpine ls
README.md
lope
lope.go
```

```
$ lope alpine cat /etc/issue
Welcome to Alpine Linux 3.7 Kernel \r on an \m (\l)
```

Mounts ~/.vault-token and forwards `VAULT_ADDR` environment variable
```
$ lope vault vault read secret/test/hellope
Key                 Value
---                 -----
refresh_interval    768h
value               world
```

Mounts ~/.kube/ for easy kubectl access
```
$ lope lachlanevenson/k8s-kubectl kubectl get pods
NAME                    READY     STATUS    RESTARTS   AGE
nginx-7c87f569d-5zvx4   1/1       Running   0          13s
```

Mounts the docker socket
```
$ lope docker docker ps
CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS                  PORTS               NAMES
b4b7bf56655b        lope                "/bin/sh -c 'docker …"   1 second ago        Up Less than a second                       elegant_lalande
```

Automagically forwards your ssh agent into the container (even on OSX!)
```
$ lope alpine/git ssh -T git@github.com
Hi Crazybus! You've successfully authenticated, but GitHub does not provide shell access.
```

## Features

### Planned

* Use host network so can talk to local host
* Add option to specify custom docker flags
* Make sure all images/names are unique so multiple lopes can be run at the same time
* If using addMount add all .dot directories instead of mounting them
* Automatically expose ports from Dockerfile
* Add yaml file to define configuration instead of doing a big one liner
* Add option in yaml file to specify mounted files
* Add yaml file option to include/exclude environment variables with pattern support
* Allow running multiple images/commands combos with stages
* Allow sharing artifacts/files between stages
* Add default .dockerignore for things like .git and .vagrant

### Done

* Write output in realtime
* Run a docker container with current directory added
* Forward all environment variables into the container
* Add blacklist and whitelist regexes to filter environment variables
* Mount secret files and directories into the container (like ~/.vault-token)
* Automatically add well known locations for secrets for development usage
* Add simple build step options. E.g. alpine as base image with a single `RUN apk add package`. 
* Add option to use bind mounts for adding the current working directory. Also allow disabling mounting current directory altogether for use cases like `vault status`
* Add command line flags
* Mount the docker socket into the container
* Automated ssh agent forwarding for OSX. https://github.com/uber-common/docker-ssh-agent-forward
* Run as current user and group when bind mounting
