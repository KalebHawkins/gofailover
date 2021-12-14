# DeviceWISE Failover Documentation

- [DeviceWISE Failover Documentation](#devicewise-failover-documentation)
  - [DeviceWISE Failover Servers](#devicewise-failover-servers)
  - [DeviceWISE Cluster Health Checks](#devicewise-cluster-health-checks)
    - [Go-Failover DeviceWISE](#go-failover-devicewise)
    - [DeviceWISE Automated Failover](#devicewise-automated-failover)
    - [DeviceWISE Manual Failover](#devicewise-manual-failover)

## DeviceWISE Failover Servers

DeviceWISE failover will be performed every 1st Sunday of the month from the primary (MASTER) node to the secondary (SLAVE) node.
Every subsequent Sunday the cluster will be checked to verify that the expected primary node is the currently running primary node of the cluster.

The failover process is performed by running the following command from the primary (MASTER) node.

```bash
pcs resource move dwgrp
pcs resource clear dwgrp
```

## DeviceWISE Cluster Health Checks

There are health checks performed for cluster before and after a failover.  

If health checks fail before a failover is triggered then a failover will ***NOT*** be attempted. An email is sent with an error message along with a summary of the cluster status for quick review.
If health checks fail after a failover an email is sent with an error message along with a summary of the cluster status for quick review.

Health checks include the following logic. 

* Are all nodes in the cluster in an `online` state?
  * Nodes should NOT be in a `standby`, `maintaince`, `pending`, `unclean`, or `shutdown` state. 
* Are all resources in the cluster in an `active` state?
  * Resources should NOT be in a `blocked` or `failed` state.
* Is there a primary (MASTER) node active?
  * For DeviceWISE is the checked by looking at the node currently running the `dwgrp` pacemaker resource.


```go
// Check to see if a node is in a healthy state.
// Code from `${PROJECT_ROOT}/cmd/helpers.go` file 
func isNodeHealthy(n crm.Node) bool {
	return n.Online && !n.Standby && !n.Maintenance && !n.Pending && !n.Unclean && !n.Shutdown
}
```

```go
// Check to see if all resources are in an active state.
// Code from `${PROJECT_ROOT}/cmd/helpers.go` file 
// ...
  for _, r := range cs.Resources.StandAlone {
      if !r.Active && r.Blocked && r.Failed {
        return fmt.Errorf("resource %v is not in a healthy state", r.Name)
      }
  }

  for _, g := range cs.Resources.Groups {
    for _, r := range g.Resources {
      if !r.Active && r.Blocked && r.Failed {
        return fmt.Errorf("resource %v is not in a healthy state", r.Name)
      }
    }
  }

  for _, c := range cs.Resources.Cloned {
    for _, r := range c.Resources {
      if !r.Active && r.Blocked && r.Failed {
        return fmt.Errorf("resource %v is not in a healthy state", r.Name)
      }
    }
  }
// ...
```

```go
// Code from `${PROJECT_ROOT}/cmd/DeviceWISE.go` file
func (dwc *DeviceWISECluster) getPrimaryNode() (string, error) {
	for _, rgrp := range dwc.clusterStatus.Resources.Groups {
		if rgrp.Name == "dwgrp" {
			return rgrp.Name, nil
		}
	}

	return "", fmt.Errorf("unable to find primary node in cluster. please check the cluster's health")
}
```

### Go-Failover DeviceWISE

To setup DeviceWISE failovers follow the scenario procedure below.

| Key | Function |
|---|---|
| DeviceWISE01 | Primay Node |
| DeviceWISE02 | Secondary Node | 

Create a `config.yaml` file on the MASTER node of the DeviceWISE cluster. 

An example config file can be seen below.

```yaml
targetPrimaryNode: DeviceWISE01
whatDay: "Sunday"
email:
  to:
    - "person1@domain.com"
    - "person2@domain.com"
    - "emailgroup@domain.com"
  from: "cluster_address@domain.com" # This can really be anything
  smtpHost: smtp.host.example.com
  smtpPort: 25
  subject: "Example subject"
```

### DeviceWISE Automated Failover

Once you have a configuration file in place you can create a cronjob to run the failover automation software every Sunday.

> The gofailover binary and config.yaml file should be in the `/appl/failover/` directory. If one does not exist create it.

You will need to create a script to hold the `PATH` variable that then executes the go binary. See below.

To get your current patch varible use the following command:

```bash
echo $PATH
```

Copy the contents of the previous command to the script in the `PATH` variable.

```bash
#!/bin/bash

PATH=<Contents from previous command>

/appl/failover/gofailover dw --config /appl/failover/config.yaml
```

Save the script above as `/appl/failover/run.sh` and make it executable. 

```bash
chmod u+x /appl/failover/run.sh
```

Now create the cronjob to run every Sunday.

```
 +---------------- minute (0 - 59)
 |  +------------- hour (0 - 23)
 |  |  +---------- day of month (1 - 31)
 |  |  |  +------- month (1 - 12)
 |  |  |  |  +---- day of week (0 - 6) (Sunday=0 or 7)
 |  |  |  |  |
 *  *  *  *  *  command to be executed
```

```bash
crontab -e 

0  3  *  *  0 /appl/failover/run.sh
```

### DeviceWISE Manual Failover

A technical node for the `gofailover` tool is that the 1st Sunday and subsequent Sunday failovers are hardcoded values in the source code.
Meaning that if you try to run the failover on any day other than Sunday the command will not be ran.

For Example: 

```bash
date
# Tue Jan  4 13:03:53 CST 2022

./gofailover dw --config config.yaml

# Example output
# Using config file: config.yaml
```

To run a manual failover manually I have provided and `override` switch that when used will run a failover regardless of the date or current primary node.

For Example:

```bash
./gofailover dw --config config.yaml --override
```

