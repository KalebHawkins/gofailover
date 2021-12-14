package cmd

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"net/smtp"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/KalebHawkins/gofailover/crm"
	"github.com/spf13/viper"
)

// ? May implement an interface system in a later version...
// ? right now this is an over complication of the code
// type Cluster interface {
// 	failoverCmd()
// 	getPrimaryNode() (string, error)
// 	healthCheck()
// 	handleError(error, crm.ClusterStatus)
// 	startFailover()
// 	handleSuccess()
// }

// Email Stuff
var (
	emailFrom string
	emailTo   []string
	smtpHost  string
	smtpPort  string
)

// generateDayMap is used to populate a global `dayMap` variable.
// This function is called only one in the `root.go` `rootCMD` `init()`
// function.
func generateDayMap() {
	dayMap = make(map[string][]int)

	var i int
	for i = 1; i <= daysIn(time.Now().Month()); i++ {
		newDate := time.Date(time.Now().Year(), time.Now().Month(), i, 0, 0, 0, 0, time.Local)

		if newDate.Month() != time.Now().Month() {
			break
		}

		dayMap[newDate.Weekday().String()] = append(dayMap[newDate.Weekday().String()], i)
	}
}

// getDay will return the ordinal day as an int and the weekday as a string
// based on the `t time.Time` parameter passed to it. The oridnal day would be
// which weekday of the month it is for example if it were to pass a time object where
// it were the 2nd Sunday of the month the function would return the values `2` and `Sunday`.
func getDay(t time.Time) (int, string) {

	var whichDay int
	var weekDay string

	for k, v := range dayMap {
		for i, n := range v {
			if k == t.Weekday().String() && n == t.Day() {
				whichDay = i + 1
				weekDay = k
			}
		}
	}

	return whichDay, weekDay
}

// daysIn returns the last day of the month providing how many days are in month provided.
func daysIn(m time.Month) int {
	return time.Date(time.Now().Year(), m+1, 0, 0, 0, 0, 0, time.Local).Day()
}

//execCmd will execute a system command and return the output as a string value.
// if a command is long running you can choose to stream the output of the command
// from stdout by setting `streamStdOut` to true.
func execCmd(cmd string, streamStdOut bool) string {
	_, err := exec.LookPath(strings.Split(cmd, " ")[0])

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to execute command: `%s`\ncommand: %v was not found in $PATH\n", cmd, strings.Split(cmd, " ")[0])
		os.Exit(1)
	}

	osCmd := exec.Command("bash", "-c", cmd)

	stdout, err := osCmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	err = osCmd.Start()
	if err != nil {
		panic(err)
	}

	var str string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if streamStdOut {
			fmt.Println(scanner.Text())
		}
		str += scanner.Text()
	}
	err = osCmd.Wait()

	if err != nil {
		panic(err)
	}

	return str
}

// getClusterStatus provided a reader containing the xml output from the command `crm_mon --as-xml`
// will return a `crm.ClusterStatus` object. This object will contain cluster data such as nodes, node status,
// resrouces, etc. (See `../crm/types.go`  for more information on the `crm.ClusterStatus` object.)
func getClusterStatus(r io.Reader) crm.ClusterStatus {
	scanner := bufio.NewScanner(r)

	var databytes []byte = make([]byte, 0)
	for scanner.Scan() {
		databytes = append(databytes, scanner.Bytes()...)
	}

	var cs crm.ClusterStatus
	if err := xml.Unmarshal(databytes, &cs); err != nil {
		panic(err)
	}

	return cs
}

// isNodeHealthy returns true if a cluster node is in an online status only.
// This function is called in the `isClusterHealthy` function.
func isNodeHealthy(n crm.Node) bool {
	return n.Online && !n.Standby && !n.Maintenance && !n.Pending && !n.Unclean && !n.Shutdown
}

// isClusterHealthy returns true if nodes check as healthy (see `isNodeHealthy`), and it checks to make sure all
// resources are in a good state. This function does not check the attributes of nodes. That should be does on a
// per cluster bases depending on what attributes are attached to your nodes.
func isClusterHealthy(cs crm.ClusterStatus) error {

	for _, n := range cs.Nodes {
		if !isNodeHealthy(n) {
			return fmt.Errorf("%v is in an unhealthy state", n.Name)
		}
	}

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

	return nil
}

// sendEmail sends an email message. The message is passed to the function as a string
// and sent using the provided configuration.
// Example config:
// 	email:
//     to:
//       - example@example.com
//     from: someone@example.com
//     smtpHost: smtp.example.com
//     smtpPort: 25
func sendEmail(msg string) {
	emailFrom = viper.GetString("email.from")
	emailTo = viper.GetStringSlice("email.to")
	smtpHost = viper.GetString("email.smtpHost")
	smtpPort = viper.GetString("email.smtpPort")
	subjectLine := viper.GetString("email.subject")

	if emailFrom == "" || emailTo == nil || smtpHost == "" || smtpPort == "" {
		fmt.Fprintf(os.Stderr, "email properties have not been set in the configuration file.\nNo emails will be sent out!\n")
	}

	// Message.
	mailMessage := []byte("Subject: " + subjectLine + "\r\n\r\n" + msg)

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, nil, emailFrom, emailTo, mailMessage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}
}
