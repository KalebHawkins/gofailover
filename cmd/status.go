package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/KalebHawkins/gofailover/crm"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Return the status of a cluster and its nodes",
	Long:  `Return the status of a cluster and its nodes`,
	Run: func(cmd *cobra.Command, args []string) {

		var cs crm.ClusterStatus
		// If the file flag is specified we parse our data from a test xml file.
		if file != "" {
			cs = statusFromFile(file)
			fmt.Println(cs)
		} else { // If the file flag is not enabled then we pull the cluster status from the crm_mon -fA1 --as-xml command.
			xml := execCmd("crm_mon -fA1 --as-xml", false)
			cs = getClusterStatus(strings.NewReader(xml))
			fmt.Println(cs)
		}

		// if the checkHealth flag is enabled then the cluster's nodes and resource states are checked.
		if checkHealth {
			isClusterHealthy(cs)
		}
	},
}

var file string
var checkHealth bool

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().StringVarP(&file, "file", "f", "", "file to pull status from")
	statusCmd.Flags().BoolVarP(&checkHealth, "health-check", "", false, "performs health check on the cluster")
}

// statusFromFile is a wrapper to pull the cluster status from a test xml file.
func statusFromFile(filePath string) crm.ClusterStatus {
	if _, err := os.Stat(file); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	f, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	cs := getClusterStatus(bytes.NewReader(f))

	return cs
}
