# hlf infra

## TOC

- [hlf infra](#hlf-infra)
  - [TOC](#toc)
  - [Description](#description)
    - [Start](#start)
    - [Stop and purge](#stop-and-purge)
    - [How to and examples](#how-to-and-examples)
    - [Exposed ports](#exposed-ports)
    - [Local development](#local-development)
  - [Notes](#notes)
  - [Links](#links)

## Description

Default empty infrastructure cluster to test and develop infrastructure code. 

### Start

```
docker-compose up
```

### Stop and purge

To stop environment use:

```
docker-compose stop
```

To completely stop and purge the environment use:

```
docker-compose down -v
```

### How to and examples

* Clone all  the necessary roles to **./toolbox/ansible/main/roles/**
* Prepare or clone playbook into **./toolbox/ansible/main/playbooks/**

```
docker-compose exec tool sh
```

```
ansible -m ping all
```

```
ansible-inventory --graph
```

```
ansible-playbook ./playbooks/main.yml
```


Connect via ssh to the **test-orderer-001.org0**
```
ssh -F .ssh/config test-orderer-001.org0
```

### Exposed ports

Full list of exposed ports and services you can find in 

[.env](.env)

with pattern **EXP_**

### Local development

You can put your custom environment variables files

```
. ./my-custom-development-variables && docker-compose up
```

Change to a stub image of service that you develop local (on your host system) for example robot

```
export IMG_BASE_SYSTEM=custom-base-system:latest
```

Full example is: 

```
export IMG_BASE_SYSTEM=custom-base-system:latest && docker-compose up
```

Or fill free to setup your custom development settings file

## Notes

* System ready to use in pipelines for integration and chaos testing and infrastructure development

## Links

* No