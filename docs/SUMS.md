# SUMS Failover Documentation

- [SUMS Failover Documentation](#sums-failover-documentation)
  - [SUMS Failover Database Servers](#sums-failover-database-servers)
  - [SUMS Database Cluster Health Checks](#sums-database-cluster-health-checks)
    - [Go-Failover SUMS](#go-failover-sums)
    - [SUMS Automated Failover](#sums-automated-failover)
    - [SUMS Manual Failover](#sums-manual-failover)

## SUMS Failover Database Servers

SUMS failover will be performed every 1st Sunday of the month from the primary (MASTER) node to the secondary (SLAVE) node.
Every subsequent Sunday the cluster will be checked to verify that the expected primary node is the currently running primary node of the cluster.

The failover process is performed by running the following command from the primary (MASTER) database node.

```bash
pg-rex_switchover
```

## SUMS Database Cluster Health Checks

There are health checks performed for the database server cluster before and after a failover.  

If health checks fail before a failover is triggered then a failover will ***NOT*** be attempted.  
If health checks fail after a failover an email is sent with an error message along with a summary of the cluster status for quick review.

Health checks include the following logic. 

* Are all nodes in the cluster in an `online` state?
  * Nodes should NOT be in a `standby`, `maintaince`, `pending`, `unclean`, or `shutdown` state. 
* Are all resources in the cluster in an `active` state?
  * Resources should NOT be in a `blocked` or `failed` state.
* Is there a primary (MASTER) node active?
  * For SUMS is the checked by looking at the `pgsql-status` attributes on the cluster nodes.


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
// Code from `${PROJECT_ROOT}/cmd/SUMS.go` file
func (sc *SUMSCluster) getPrimaryNode() (string, error) {
	for _, catrs := range sc.clusterStatus.Attributes {
		for _, atr := range catrs.Attributes {
			if atr.Name == "pgsql-status" && atr.Value == "PRI" {
				return catrs.Node, nil
			}
		}
	}

	return "", fmt.Errorf("unable to find primary node in cluster. please check the cluster's health")
}
```

### Go-Failover SUMS

To setup SUMS database failovers follow the scenario procedure below.

| Key | Function |
|---|---|
| SUMSdb01 | Primay Node (MASTER) |
| SUMSdb02 | Secondary Node (SLAVE) | 

Create a `config.yaml` file on the MASTER node of the SUMS database cluster. 

An example config file can be seen below.

```yaml
targetPrimaryNode: SUMSdb01
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

### SUMS Automated Failover

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

/appl/failover/gofailover sums --config /appl/failover/config.yaml
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

### SUMS Manual Failover

A technical node for the `gofailover` tool is that the 1st Sunday and subsequent Sunday failovers are hardcoded values in the source code.
Meaning that if you try to run the failover on any day other than Sunday the failover command will not ebe executed.

For Example: 

```bash
date
# Tue Jan  4 13:03:53 CST 2022

./gofailover sums --config config.yaml

# Example output
# Using config file: config.yaml
```

To run a manual failover manually I have provided and `override` switch that when used will run a failover regardless of the date or current primary node.

For Example:

```bash
./gofailover SUMS --config config.yaml --override
```

