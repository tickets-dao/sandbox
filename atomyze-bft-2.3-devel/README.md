# Atomyze bft 2.3 devel 

## TOC

- [Atomyze bft 2.3 devel](#atomyze-bft-23-devel)
  - [TOC](#toc)
  - [Description](#description)
  - [How to](#how-to)
    - [Start](#start)
    - [Help](#help)
    - [Force reload configuration](#force-reload-configuration)
    - [Stop and purge](#stop-and-purge)
    - [Exposed ports](#exposed-ports)
    - [Custom tools in sandbox](#custom-tools-in-sandbox)
    - [Local development](#local-development)
      - [Auto-apply configuration](#auto-apply-configuration)
    - [Backup and restore](#backup-and-restore)
    - [Namespaces channels and chaincodes](#namespaces-channels-and-chaincodes)
      - [Chaincode from existing package](#chaincode-from-existing-package)
    - [Certificates and connection.yml](#certificates-and-connectionyml)
    - [Run-on](#run-on)
    - [Grafana and custom dashboards](#grafana-and-custom-dashboards)
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
tool_1                    | -- INFO: consistent state 3a60e7a068835f47aeee506cfd974adf  /state/.hash
```

Every time you see state hash in log mean that system successfully reach consistent condition. 
And ready to process requests.

### Help

To get all supported tool list and available internal function

```
docker-compose exec tool --help
```

### Force reload configuration

To force initiate configuration reload use reload script inside tool container

```
docker-compose exec tool reload
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

### Exposed ports

Full list of exposed ports and services you can find in 

[.env](.env)

with pattern **EXP_**

### Custom tools in sandbox

To run additional utilities such as workload generation utilities, you can use this approach

* Add custom docker-compose file with necessary tool and bind mounе **tool state** volume and data directory for **data/out**

Example for l2:

```
version: '3.4'

services:
  l2:
    image: registry.n-t.io/atmz/l2/loader2:1.0.0
    command: generate_load 10tps
    restart: unless-stopped
    volumes:
      - tool:/state
      - ./tool/data:/data
```

* Start compose with custom docker-compose file

```
docker-compose -f l2.yml -f docker-compose.yml up
```

### Local development

For local development e.g. robot or observer you need override fabric specific gossip variables:

```
. ./env-local-development && docker-compose up
```

Or you can put your custom environment variables files instead of **env-local-development**

```
. ./my-custom-development-variables && docker-compose up
```

Than change to a stub image of service that you develop local (on your host system) for example robot

```
export IMG_ROBOT=alpine:3
```

Full example is: 

```
export IMG_ROBOT=alpine:3 && . ./env-local-development && docker-compose up
```

Or fill free to setup your custom development settings file

#### Auto-apply configuration

To control auto apply loop you can setup check timeout before **up** environment


```
export SLEEP_STATE=30 # sleep to check changes
export SLEEP_ERROR=30 # sleep after error appears 
```

To disable auto apply configuration (oneshot start)

```
export SLEEP_STATE=9999
```

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
To setup policy and version of chaincode use **.prepare** file

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

Example for public namespace with 3 organizations **without initialization**:

```
CHAINCODE_VERSION="$(date +%s)"
CHAINCODE_POLICY="AND('org0.member', OutOf(1, 'org1.member', 'org2.member'))"
```

Example for private namespace with 2 organizations:

```
CHAINCODE_VERSION="$(date +%s)"
CHAINCODE_POLICY="AND('org0.member', 'org2.member')"
```

Example with manual initialization later:

```
CHAINCODE_VERSION="$(date +%s)"
CHAINCODE_POLICY="AND('org0.member')"
CHAINCODE_INIT="skip"
```

Example with initialization and internal helper function usage

```
CHAINCODE_VERSION="$(date +%s)"
CHAINCODE_POLICY="AND('org0.member')"

key0="$(_crypto_admin_key_by_org "org0")"
key1="$(_crypto_admin_key_by_org "org1")"
ski0="$(_tool_ski_by_private_key "$key0")"
ski1="$(_tool_ski_by_private_key "$key1")"

CHAINCODE_INIT="{\"Args\":[\"$ski0\",\"1\",\"$ski1\"]}"
```

This **.prepare** example contains the standard channel policy and allows you to automatically change the version of the chaincodes, which is very useful for local development


If you place **.prepare** at the namespace level, it will be executed after successfully reaching a consistent state. This is useful for various tests and initial data fills.

```
./tool/data/channel/
├── .gitignore
├── private
│   └── system
│       └── configtx.yaml
└── public
    ├── .prepare  <--------------------
    └── system
        └── configtx.yaml
```

#### Chaincode from existing package

The system allows you to install chaincodes from packages
Just put package tar.gz file with to channel directory
```
./tool/data/channel/
├── private
│   └── system
│       └── configtx.yaml
└── public
    ├── acl
    │   ├── acl.tar.gz <-----------------
    │   └── acl.tar.gz.prepare
    └── system
        └── configtx.yaml

```

To setup policy/initialization for chaincodes in channel use **.prepare** file on a channel level.  
To setup policy/initialization for specific **example.tar.gz** place .prepare content to **example.tar.gz.prepare** file

* CHAINCODE_VERSION - is taken from the package file and ignored in .prepare

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
│   │       └── User1@org0
│   │           ├── ca
│   │           ├── connection.yaml <------
│   └── public
│       └── org0
│           └── User1@org0
│               ├── ca
│               ├── connection.yaml <------

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

### Grafana and custom dashboards

To setup custom grafana dashboards just put dashboard json file into **grafana/data/dashboard** directory

```
./grafana/data/
├── dashboard <------------
│   └── .gitkeep
└── .prepare

```

Dashboards json will automatically import into running grafana instance.
All internal datasources are configured automatically.

### Host metrics

When the system starts up, the internal prometheus starts polling http://localhost:8080/metrics of the host machine.  
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
