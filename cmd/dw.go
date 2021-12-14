package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/KalebHawkins/gofailover/crm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type DeviceWISECluster struct {
	expectedPrimaryNode string
	currentPrimaryNode  string
	clusterStatus       crm.ClusterStatus
}

// DeviceWISECluster.failoverCmd() runs the commands to preform the failover for DeviceWISE nodes.
func (dwc *DeviceWISECluster) failoverCmd() {
	// Moving the PCS resource indirectly creates a location constraint on the resource.
	// So we need to make sure that once the resource is moved we clear that location constraint.
	fmt.Println("Running: pcs resource move dwgrp")
	execCmd("pcs resource move dwgrp", false)

	// There needs to sleep time between clearing the resource constriants
	// and moving the resource group to the other node. If this timer isn't here the code
	// will run too fast for the resources to actually move before the resource containts are cleared
	// out.
	time.Sleep(25 * time.Second)

	fmt.Println("Running: pcs resource clear dwgrp")
	execCmd("pcs resource clear dwgrp", false)

	// This section performs a confirmation that the location consttraints were removed. If they
	// were not removed as intended then we flag an email to be sent and exit.
	fmt.Println("Confirming location constraints were removed")
	if results := execCmd("pcs constraint location", false); strings.Contains(results, "Node:") {
		dwc.handleError(
			fmt.Errorf("failed to clear location constraints:\n%v\n\nPlease login to one of the cluster nodes and run `pcs resource clear dwgrp` manually to attempt to clear constraints", results),
			dwc.clusterStatus)
	}
}

// DeviceWISECluster.getPrimaryNode() returns the cluster's current primary node by looking at which node
// is currently running the `dwgrp` resource group. An error is returned if that node isn't found.
func (dwc *DeviceWISECluster) getPrimaryNode() (string, error) {
	for _, rgrp := range dwc.clusterStatus.Resources.Groups {
		if rgrp.Name == "dwgrp" {
			return rgrp.Resources[0].Node.Name, nil
		}
	}

	return "", fmt.Errorf("unable to find primary node in cluster. please check the cluster's health")
}

// DeviceWISECluster.handleError() simply sends and email out containing the passed parameters' information if an error occurs.
// This function does perform a call os.Exit(1) meaning no further execution will take place.
func (dwc *DeviceWISECluster) handleError(err error, cs crm.ClusterStatus) {
	msg := `There was an error encounter when attempting to perform a failover on the DeviceWISE nodes.
Failover procedures will not be performed until this is corrected. Please see the erorr message below along with the cluster status.

Error Message:
%v

Cluster Status:
%v
`
	sendEmail(fmt.Sprintf(msg, err, cs))
	os.Exit(1)
}

// DeviceWISECluster.handleSuccess() sends an email upon successful failover.
func (dwc *DeviceWISECluster) handleSuccess() {
	msg := `
Failover procedure completed without detected errors.
Please see the output below to verify that the cluster looks healthy.

Current Primary Node: %v

Cluster Status: 
%v
`
	sendEmail(fmt.Sprintf(msg, dwc.currentPrimaryNode, dwc.clusterStatus))
}

// DeviceWISECluster.healthcheck() will make calls to DeviceWISE.handleError() if an error occurs during execution.
// Cluster health checks are performed by checking that all nodes are in a healthy state and all resources are in a
// active state.
func (dwc *DeviceWISECluster) healthCheck() {
	status := execCmd("crm_mon -fA1 --as-xml", false)
	cs := getClusterStatus(strings.NewReader(status))

	err := isClusterHealthy(cs)

	if err != nil {
		dwc.handleError(err, cs)
	}

	dwc.clusterStatus = cs
	dwc.currentPrimaryNode, err = dwc.getPrimaryNode()

	if err != nil {
		dwc.handleError(err, cs)
	}
}

// DeviceWISE.startFailover() performs all the required actions to perform the failover on DeviceWISE nodes.
func (dwc *DeviceWISECluster) startFailover() {
	dwc.expectedPrimaryNode = viper.GetString("targetPrimaryNode")

	if dwc.expectedPrimaryNode == "" {
		fmt.Fprintln(os.Stderr, "`targetPrimaryNode` is not set in the configuration file")
		os.Exit(1)
	}

	if override {
		dwc.healthCheck()
		dwc.failoverCmd()
		dwc.healthCheck()
		dwc.handleSuccess()
		return
	}

	ordinalDay, weekDay := getDay(time.Now())
	whatWeekDay := viper.GetString("whatDay")
	if whatWeekDay == "" {
		whatWeekDay = "Sunday"
	}

	whatWeekDay = strings.Title(whatWeekDay)

	if ordinalDay == 1 && weekDay == whatWeekDay {
		dwc.healthCheck()

		if dwc.currentPrimaryNode == dwc.expectedPrimaryNode {
			dwc.failoverCmd()
			dwc.healthCheck()
			dwc.handleSuccess()
		}
	} else if ordinalDay != 1 && weekDay == whatWeekDay {
		dwc.healthCheck()

		if dwc.currentPrimaryNode != dwc.expectedPrimaryNode {
			dwc.failoverCmd()
			dwc.healthCheck()
			dwc.handleSuccess()
		}
	}
}

// dwCmd represents the dw command
var dwCmd = &cobra.Command{
	Use:   "dw",
	Short: "Command used to perform DB failover for DeviceWISE systems.",
	Long:  `Command used to perform DB failover for DeviceWISE systems.`,
	Run: func(cmd *cobra.Command, args []string) {
		deviceWiseCluster := DeviceWISECluster{}
		deviceWiseCluster.startFailover()
	},
}

func init() {
	rootCmd.AddCommand(dwCmd)
}
