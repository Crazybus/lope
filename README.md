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
* Defaults which favour usability over speed and security while still having configuration options available to restrict which environment variables/secrets are forwarded into containers for proper usage. 
* All actions are immutable. All files and dependencies are added to a docker image before running. This means no local state is modified and the state inside the container is static during its lifetime. This makes it possible to run something like `ansible-playbook` and then immediately switch to a different branch and continue to make changes.
* Testing and tooling should be run the same in development and CI environments

## Command line flags
| Flag                   | Effect                                                                                                                                           |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| -addDocker             | Uses wget to download the docker client binary into the image (default `false`)                                                                  |
| -addMount              | Setting this will add the directory into the image instead of mounting it                                                                        |
| -blacklist _string_    | Comma seperated list of environment variables that will be ignored by lope (default `"HOME,SSH_AUTH_SOCK,TMPDIR"`)                               |
| -dir _string_          | The directory that will be mounted into the container. Defaut is current working directory                                                       |
| -dockerSocket _string_ | Path to the docker socket (default `"/var/run/docker.sock"`)                                                                                     |
| -entrypoint _string_   | The entrypoint for running the lope command (default `/bin/sh`)                                                                                  |
| -instruction _value_   | Extra docker image instructions to run when building the image. Can be specified multiple times                                                  |
| -noDocker              | Disables mounting the docker socket inside the container (default `false`)                                                                       |
| -noRoot                | Use current user instead of the root user (default `false`)                                                                                      |
| -noTty                 | Disable the --tty flag (default `false`)                                                                                                         |
| -noMount               | Disable mounting the current working directory into the image                                                                                    |
| -noSSH                 | Disable forwarding ssh agent into the container                                                                                                  |
| -path _value_          | Paths that will be mounted from the users home directory into lope. Path will be ignored if it isn't accessible. Can be specified multiple times |
| -workDir  _string_     | The default working directory for the docker image (default `"/lope"`)                                                                           |
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

Attach into the container for some interactive debugging fun
```
$ lope alpine sh
/lope #
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

Start a web server and connect to it from another lope command
```
$ lope python python3 -m http.server
Serving HTTP on 0.0.0.0 port 8000 (http://0.0.0.0:8000/) ...

# In another terminal
$ lope python curl -I localhost:8000
HTTP/1.0 200 OK
Server: SimpleHTTP/0.6 Python/3.6.5
Date: Sat, 02 Jun 2018 19:42:42 GMT
Content-type: text/html; charset=ascii
Content-Length: 764
```

Run the unit tests for [phpunit](https://github.com/sebastianbergmann/phpunit)
```
$ lope composer 'composer install && ./phpunit'
OK, but incomplete, skipped, or risky tests!
Tests: 1857, Assertions: 3206, Skipped: 13.
```

Add docker client to a random docker image (requires wget to be in the image)
```
$ lope -addDocker alpine docker ps
CONTAINER ID        IMAGE                           COMMAND                  CREATED             STATUS                  PORTS                   NAMES
bf8d6885a2de        lope                            "/bin/sh -c 'docker …"   1 second ago        Up Less than a second                           elegant_villani
```

Run the kitchen docker tests for the ansible role [ansible-elasticsearch](https://github.com/elastic/ansible-elasticsearch)
```
lope -workDir /lope/elasticsearch \
     -addDocker \
     ruby:2.3-onbuild \
     bundle exec kitchen converge standard-ubuntu-1604
```

Run ansible against a host that uses ssh to authenticate
```
$ lope williamyeh/ansible:alpine3 ansible all -i 'lope-host,' -m shell -a 'hostname'
lope-host | SUCCESS | rc=0 >>
debian-9
```

## Features

### Planned

* Add option to specify custom docker flags
* Get vagrant/virtualbox combo working
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

* Add support for attaching into container for debugging
* Use host network by default so containers can talk to localhost and other containers
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
