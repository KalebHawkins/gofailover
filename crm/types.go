package crm

import (
	"fmt"
	"reflect"
)

// ClusterStatus contains a summary of the cluster status, cluster nodes, attributes of those nodes, and resources.
// The status is parsed from the `crm_mon -fA1 --as-xml` command output.
type ClusterStatus struct {
	Status     Summary     `xml:"summary"`
	Nodes      []Node      `xml:"nodes>node"`
	Attributes []Attribute `xml:"node_attributes>node"`
	Resources  Resources   `xml:"resources"`
}

func (cs ClusterStatus) String() string {
	var str string
	str += cs.Status.String() + "\n"

	for _, n := range cs.Nodes {
		str += n.String() + "\n"

		for _, a := range cs.Attributes {
			if a.Node == n.Name && a.Attributes != nil {
				str += a.String()
			}
		}
	}

	str += "\nResources Summary:\n"

	for _, n := range cs.Nodes {
		str += fmt.Sprintf("  Node: %v\n", n.Name)

		for _, sr := range cs.Resources.StandAlone {
			if sr.Node.Name == n.Name {
				str += sr.String()
			}
		}

		for _, g := range cs.Resources.Groups {
			str += fmt.Sprintf("    Group: %v\n", g.Name)
			for _, gr := range g.Resources {
				if gr.Node.Name == n.Name {
					str += gr.String()
				}
			}
		}

		for _, c := range cs.Resources.Cloned {
			str += fmt.Sprintf("    Clone: %v\n", c.Name)
			for _, cr := range c.Resources {
				if cr.Node.Name == n.Name {
					str += cr.String()
				}
			}
		}
	}

	return str
}

// Summary contains general status of the cluster.
type Summary struct {
	Stack struct {
		Type string `xml:"type,attr"`
	} `xml:"stack"`
	DesignatedController struct {
		Node   string `xml:"name,attr"`
		Quorum bool   `xml:"with_quorum,attr"`
	} `xml:"current_dc"`
	NodesConfigured struct {
		Number int `xml:"number,attr"`
	} `xml:"nodes_configured"`
	ResourcesConfigured struct {
		Number int `xml:"number,attr"`
	} `xml:"resources_configured"`
	Options struct {
		StonithEnabled   bool   `xml:"stonith-enabled,attr"`
		SymmetricCluster bool   `xml:"symmetric-cluster,attr"`
		NoQuorumPolicy   string `xml:"no-quorum-policy,attr"`
		MaintenanceMode  bool   `xml:"maintenance-mode,attr"`
	} `xml:"cluster_options"`
}

func (s Summary) String() string {
	str := `Status Summary:
  Stack Type           : %v
  Designated Controller: [ Node: %v | HasQuorum: %v ]
  Nodes Configured     : %v
  Resources Configured : %v
  Cluster Options      : [ Stonith Enabled: %v | Symmetric Cluster: %v | No Quorum Policy: %v | Maintenance Mode: %v ] 
`

	str = fmt.Sprintf(str, s.Stack.Type,
		s.DesignatedController.Node, s.DesignatedController.Quorum,
		s.NodesConfigured.Number, s.ResourcesConfigured.Number, s.Options.StonithEnabled,
		s.Options.SymmetricCluster, s.Options.NoQuorumPolicy, s.Options.MaintenanceMode)

	return str
}

// Node contains cluster node status information.
type Node struct {
	Name        string `xml:"name,attr"`
	Online      bool   `xml:"online,attr"`
	Standby     bool   `xml:"standby,attr"`
	Maintenance bool   `xml:"maintenance,attr"`
	Pending     bool   `xml:"pending,attr"`
	Unclean     bool   `xml:"unclean,attr"`
	Shutdown    bool   `xml:"shutdown,attr"`
}

func (n Node) String() string {
	str := fmt.Sprintf("Node: %s\n", n.Name)

	str += fmt.Sprintln("  Status:")
	v := reflect.ValueOf(n)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Type().Name() == "bool" && v.Field(i).Interface() == true {
			str += fmt.Sprintf("    %s: %v", v.Type().Field(i).Name, v.Field(i).Interface())
		}
	}

	return str
}

// Attribute is a struct containing a
// The Node property is the node that contains the attributes.
type Attribute struct {
	Node       string `xml:"name,attr"`
	Attributes []struct {
		Name  string `xml:"name,attr"`
		Value string `xml:"value,attr"`
	} `xml:"attribute"`
}

func (a Attribute) String() string {
	str := "  Attributes:\n"
	for _, attr := range a.Attributes {
		str += fmt.Sprintf("    %v: %v\n", attr.Name, attr.Value)
	}

	return str
}

// Resources contains collections of standalone resources, grouped resources, and cloned resources.
type Resources struct {
	StandAlone []StandAloneResource `xml:"resource"`
	Groups     []ResourceGroup      `xml:"group"`
	Cloned     []ResourceClone      `xml:"clone"`
}

// StandAloneResource is a structure containing resources of the cluster that are not grouped or cloned.
// This structure contains not only various resource properties like the status and agent but also the node
// that the resource is running on.
type StandAloneResource struct {
	Node struct {
		Name string `xml:"name,attr"`
	} `xml:"node"`
	Name    string `xml:"id,attr"`
	Agent   string `xml:"resource_agent,attr"`
	Role    string `xml:"role,attr"`
	Active  bool   `xml:"active,attr"`
	Blocked bool   `xml:"blocked,attr"`
	Managed bool   `xml:"managed,attr"`
	Failed  bool   `xml:"failed,attr"`
}

func (r StandAloneResource) String() string {
	fmtString := "      [ Name: %v | Agent: %v | Role: %v | Active: %v | Blocked: %v | Managed: %v | Failed: %v ]\n"

	str := fmt.Sprintf(fmtString, r.Name, r.Agent, r.Role, r.Active, r.Blocked, r.Managed, r.Failed)

	return str
}

// ResourceGroup contains the resource group name along with a collection of GroupedResources.
type ResourceGroup struct {
	Name      string            `xml:"id,attr"`
	Resources []GroupedResource `xml:"resource"`
}

// GroupedResource contains the same general structure as a StandAloneResource.
type GroupedResource struct {
	Node struct {
		Name string `xml:"name,attr"`
	} `xml:"node"`
	Name    string `xml:"id,attr"`
	Agent   string `xml:"resource_agent,attr"`
	Role    string `xml:"role,attr"`
	Active  bool   `xml:"active,attr"`
	Blocked bool   `xml:"blocked,attr"`
	Managed bool   `xml:"managed,attr"`
	Failed  bool   `xml:"failed,attr"`
}

func (r GroupedResource) String() string {
	fmtString := "      [ Name: %v | Agent: %v | Role: %v | Active: %v | Blocked: %v | Managed: %v | Failed: %v ]\n"

	str := fmt.Sprintf(fmtString, r.Name, r.Agent, r.Role, r.Active, r.Blocked, r.Managed, r.Failed)

	return str
}

// ResourceClone contains the resource clone name along with a collection of ClonedResources.
type ResourceClone struct {
	Name      string           `xml:"id,attr"`
	Resources []ClonedResource `xml:"resource"`
}

// ClonedResource contains the same general structure as a StandAloneResource.
type ClonedResource struct {
	Node struct {
		Name string `xml:"name,attr"`
	} `xml:"node"`
	Name    string `xml:"id,attr"`
	Agent   string `xml:"resource_agent,attr"`
	Role    string `xml:"role,attr"`
	Active  bool   `xml:"active,attr"`
	Blocked bool   `xml:"blocked,attr"`
	Managed bool   `xml:"managed,attr"`
	Failed  bool   `xml:"failed,attr"`
}

func (r ClonedResource) String() string {
	fmtString := "      [ Name: %v | Agent: %v | Role: %v | Active: %v | Blocked: %v | Managed: %v | Failed: %v ]\n"

	str := fmt.Sprintf(fmtString, r.Name, r.Agent, r.Role, r.Active, r.Blocked, r.Managed, r.Failed)

	return str
}
