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

type SUMSCluster struct {
	expectedPrimaryNode string
	currentPrimaryNode  string
	clusterStatus       crm.ClusterStatus
}

// SUMSCluster.failoverCmd() runs the command to perform the failover for SUMS database nodes.
func (sc *SUMSCluster) failoverCmd() {
	execCmd("yes | pg-rex_switchover", true)
}

// SUMSCluster.getPrimaryNode() returns the cluster's current primary node by looking at the pgsql-status attribute of the nodes.
// If there is no primary found then an error is returned. Not to mention something is wrong with the cluster and it should be investigated.
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

// SUMSCluster.handleError() simply sends and email out containing the passed parameters' information if an error occurs.
// This function does perform a call os.Exit(1) meaning no further execution will take place.
func (sc *SUMSCluster) handleError(err error, cs crm.ClusterStatus) {
	msg := `There was an error encounter when attempting to perform a failover on the SUMS database nodes.
Failover procedures will not be performed until this is corrected. Please see the error message below along with the cluster status.

Error Message:
%v

Cluster Status:
%v
`
	sendEmail(fmt.Sprintf(msg, err, cs))
	os.Exit(1)
}

// SUMSCluster.handleSuccess() sends an email upon successful failover.
func (sc *SUMSCluster) handleSuccess() {
	msg := `
Failover procedure completed without detected errors.
Please see the output below to verify that the cluster looks healthy.

Current Primary Node: %v

Cluster Status: 
%v
`
	sendEmail(fmt.Sprintf(msg, sc.currentPrimaryNode, sc.clusterStatus))
}

// SUMSCluster.healthCheck() will make a call to SUMSCluster.handleError() if there are cluster health issues.
// If all checks pass this function will set the SUMSCluster instance's clusterStatus property.
// This function also makes a call to SUMSCluster.getPrimaryNode to set the SUMSCluster.currentPrimaryNode property
// active in the cluster. If there is not that is a clear sign the cluster is not in a healthy state and another call to
// SUMSCluster.handleError() is made.
func (sc *SUMSCluster) healthCheck() {
	status := execCmd("crm_mon -fA1 --as-xml", false)
	cs := getClusterStatus(strings.NewReader(status))

	err := isClusterHealthy(cs)

	if err != nil {
		sc.handleError(err, cs)
	}

	sc.clusterStatus = cs

	sc.currentPrimaryNode, err = sc.getPrimaryNode()

	if err != nil {
		sc.handleError(err, cs)
	}
}

// SUMSCluster.startFailover() performs all the required actions to perform the failover on SUMS database nodes.
func (sc *SUMSCluster) startFailover() {
	// Get what node should be considered the primary node of the cluster from the
	// provided configuration file. If the `targetPrimaryNode` is not in the configuration
	// and error is displayed and the software exits.
	sc.expectedPrimaryNode = viper.GetString("targetPrimaryNode")
	if sc.expectedPrimaryNode == "" {
		fmt.Fprintln(os.Stderr, "`targetPrimaryNode` is not set in the configuration file")
		os.Exit(1)
	}

	// If the override switch is flipped on then perform a failover regardless of the day
	// of the week or which node is the current primary. This will not run if health checks fail.
	if override {
		sc.healthCheck()
		sc.failoverCmd()
		sc.healthCheck()
		sc.handleSuccess()
		return
	}

	// Get the current ordinal and weekday. For example 1st of Sunday month would be
	// returned as 1 Sunday.
	ordinalDay, weekDay := getDay(time.Now())
	whatWeekDay := viper.GetString("whatDay")
	if whatWeekDay == "" {
		whatWeekDay = "Sunday"
	}
	whatWeekDay = strings.Title(whatWeekDay)

	// If it is the 1st Sunday of the month perform health checks
	if ordinalDay == 1 && weekDay == whatWeekDay {
		sc.healthCheck()

		// Only if the currently running primary node is the expected primary node do we perform the failover
		if sc.currentPrimaryNode == sc.expectedPrimaryNode {
			sc.failoverCmd()
			sc.healthCheck()
			sc.handleSuccess()
		}
		// Otherwise if it is any other Sunday we attempt to fail back to the expected primary node.
	} else if ordinalDay != 1 && weekDay == whatWeekDay {
		sc.healthCheck()

		if sc.currentPrimaryNode != sc.expectedPrimaryNode {
			sc.failoverCmd()
			sc.healthCheck()
			sc.handleSuccess()
		}
	}
}

// sumsCmd represents the SUMS command
var sumsCmd = &cobra.Command{
	Use:   "sums",
	Short: "Command used to perform DB failover for SUMS systems.",
	Long:  `Command used to perform DB failover for SUMS systems.`,
	Run: func(cmd *cobra.Command, args []string) {
		sumsCluster := SUMSCluster{}
		sumsCluster.startFailover()
	},
}

func init() {
	rootCmd.AddCommand(sumsCmd)
}
