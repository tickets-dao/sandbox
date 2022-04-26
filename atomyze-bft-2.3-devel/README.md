# Atomyze bft 2.3 devel 

## TOC

- [Atomyze bft 2.3 devel](#atomyze-bft-23-devel)
  - [TOC](#toc)
  - [Description](#description)
  - [How to](#how-to)
    - [Start](#start)
    - [Stop and purge](#stop-and-purge)
    - [Exposed ports](#exposed-ports)
    - [Local development](#local-development)
    - [Backup and restore](#backup-and-restore)
    - [Namespaces channels and chaincodes](#namespaces-channels-and-chaincodes)
    - [Certificates and connection.yml](#certificates-and-connectionyml)
    - [Run-on](#run-on)
    - [Host metrics](#host-metrics)
    - [Invoke and query](#invoke-and-query)
    - [Switch fabric version](#switch-fabric-version)
  - [Notes](#notes)
  - [Links](#links)

## Description

Atomyze full feature development with hlf bft 2.3 and off-chain component.
Environment ready to use and suitable for local development and testing.
The fabric configuration is described in declarative terms using the file structure.  
The system automatically tracks changes in the configuration directory and applies them to the state.  
The system provides additional utilities for state analysis and development:

* prometheus - collects internal metrics from all internal services
* grafana - display metrics and provides alerting system
* mailhog - fake local mta and mail client to display alerts 

Default service users: 
- admin
- test

Default password:
- test

The system additionally provides services:
- postgres
- redis

All chaincodes are run in isolated container inside dind(docker in docker)
It is highly discouraged to change the **IP** variable for security reasons

## How to

### Start

```
docker-compose up
```

Wait until the system reaches a consistent condition. 
This can be determined by recording a hash of the state in the logs.

```
tool_1                    | 176f52ae3ac92b90558e91361d8fc046  /state/.hash
```

Every time you see state hash in log mean that system successfully reach consistent condition. 
And ready to process requests.

### Stop and purge

To stop environment use:

```
docker-compose stop
```

To completely stop and purge the environment use:

```
docker-compose down -v
```

### Exposed ports

Full list of exposed ports and services you can find in 

[.env](.env)

with pattern **EXP_***

### Local development

For local development e.g. robot or observer you need override fabric specific gossip variables:

```
. ./default-local-settings && docker-compose up
```

Or you can put your custom environment variables files instead of **default-local-settings**

```
. ./my-custom-development-variables && docker-compose up
```

Than change to a stub image of service that you develop local (on your host system) for example robot

```
export IMG_ROBOT=alpine:3
```

Full example is: 

```
export IMG_ROBOT=alpine:3 && . ./default-local-settings && docker-compose up
```

Or fill free to setup your custom development settings file

### Backup and restore

To make full consistent environment backup use this steps:

* stop all services to prevent binary data corruption

```
docker-compose stop
```

* start only **tool** service

```
docker-compose up -d tool
```

* run **backup** routine inside **tool** container

```
docker-compose exec tool backup
```

* to **restore** use the same steps except last

```
docker-compose exec tool restore
```

Backup file located in **tool/data/out/backup.tar.gz** fill free to share it and make pre configured environment. 

### Namespaces channels and chaincodes

The main idea of this solution this is a declarative mapping of the file structure into the fabric configuration.

Default zero cluster configuration looks like this:

```
tool/data/channel/
├── private
│   └── system
│       └── configtx.yaml
└── public
    └── system
        └── configtx.yaml
```

* channel - root of configuration directory
* private / public - separate clusters with different configuration - so-called namespace (atomyze specific)
* system - channel name

To create new channel you need just create directory into namespace

```
tool/data/channel/
├── private
│   └── system
└── public
    ├── channel1 
    ├── channel2
    ├── channel3
    └── system
```
This will automatically **create channel1 channel2 channel3**.  
To install chaincode you just need to copy chaincod directory with code into channel directory

```
tool/data/channel/
├── private
│   └── system
│       └── configtx.yaml
└── public
    ├── channel1
    │   └── chaincode1 <---------------
    │       ├── go.mod
    │       ├── go.sum
    │       └── main.go

```

This will automatically install **chaincode1** into **channel1** into cluster/namespace **public**
To setup policy and verion of chaincode use **.prepare** file

```
tool/data/channel/
├── .gitignore
├── private
│   └── system
│       └── configtx.yaml
└── public
    ├── channel1
    │   ├── chaincode1
    │   │   ├── go.mod
    │   │   ├── go.sum
    │   │   └── main.go
    │   └── .prepare <-----------------
```
* On **channel** level it will apply to all chaincodes in this channel
* Locate **.prepare** file in chaincode directory for chaincode specific configuration

Example for public namespace with 3 organizations:

```
CHAINCODE_VERSION="$(date +%s)"
CHAINCODE_POLICY="AND('org0.peer', OutOf(1, 'org1.peer', 'org2.peer'))"
```

Example for private namespace with 2 organizations:

```
CHAINCODE_VERSION="$(date +%s)"
CHAINCODE_POLICY="AND('org0.peer', 'org2.peer')"
```

This **.prepare** example contains the standard channel policy and allows you to automatically change the version of the chaincodes, which is very useful for local development

### Certificates and connection.yml

After the system reaches a consistent state

```
tool_1                    | 176f52ae3ac92b90558e91361d8fc046  /state/.hash
```

It will automatically generate the necessary cryptomaterials and **connection.yml** for each namespace 

```
tool/data/out/
├── backup.tar.gz
├── connection
│   ├── private
│   │   └── org0
│   │       ├── ca
│   │       ├── connection.yaml <------
│   │       └── User1@org0
│   └── public
│       └── org0
│           ├── ca
│           ├── connection.yaml <------
│           └── User1@org0

```

This makes it easy to configure the fabric sdk

### Run-on

There is a **run-on** utility to manage the service in the system

```
docker-compose exec tool run-on --help
```

```
Usage: /data/bin/run-on [host] [/path/to/script]

Command copy selected shell script to remote host and execute it.

Example:
    /data/bin/run-on "test-peer-004.org1" "/data/bin/script_russian_roulette" 
    /data/bin/run-on "test-peer-001.org1" "/data/bin/script_reboot" 
    /data/bin/run-on "test-peer-002.org1" "/data/bin/script_tc_latency" 
    /data/bin/run-on "test-peer-004.org1" "/data/bin/script_tc_bad_network" 

Host:
  - connection
  - grafana
  - postgres
  - prometheus
  - redis
  - test-observer-001.org0
  - test-orderer-001.org0
  - test-orderer-001.org1
  - test-orderer-002.org0
  - test-orderer-002.org1
  - test-orderer-011.org0
  - test-orderer-011.org1
  - test-orderer-012.org0
  - test-orderer-012.org1
  - test-peer-001.org0
  - test-peer-001.org1
  - test-peer-001.org2
  - test-peer-002.org0
  - test-peer-002.org1
  - test-peer-002.org2
  - test-robot-001.org0

Script:
  - /data/bin/script_sigcont
  - /data/bin/script_sigstop
  - /data/bin/script_tc
  - /data/bin/script_reboot
```

This utility useful for chaos testing and research of the system in boundary states

### Host metrics

When the system starts up, the internal prometeus starts polling http://localhost:8080/metrics of the host machine.  
This can be useful for local development. 

### Invoke and query

For convenient chaincode invoke and query there are scripts

```
docker-compose exec tool invoke
```

and

```
docker-compose exec tool query
```

### Switch fabric version

To switch fabric version simple import environment file with version you want and than start cluster as usual 

```
. ./env-hlf-2.4.3 && docker-compose up
```

## Notes

* System ready to use in pipelines for integration and chaos testing 

## Links

* [IBM Blockchain Platform Extension for VS Code](https://marketplace.visualstudio.com/items?itemName=IBMBlockchain.ibm-blockchain-platform)
