package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var dayMap map[string][]int
var override bool

var rootCmd = &cobra.Command{
	Use:   "failover",
	Short: "Tool for failing over different systems in an automated fashion.",
	Long: `Tool for failing over different systems in an automated fashion.
Currently Supported:

- ✅ PKM/EKM
- ✅ DeviceWise
- ✅ SUMs
`,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.test.yaml)")
	rootCmd.PersistentFlags().BoolVar(&override, "override", false, "run a subcommand regardless of the current date")

	generateDayMap()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".test")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
