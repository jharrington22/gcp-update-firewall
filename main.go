package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

var creds string
var pattern string
var force bool

func init() {
	flag.StringVar(&creds, "j", "", "Path to Google auth json (If not specified default location will be assumed)")
	flag.StringVar(&pattern, "p", "", "Pattern to match when scanning firewall rule names (* not needed matches on string found anywhere in name)")
	flag.BoolVar(&force, "f", false, "Force the updating of rules where the IP is set already to a single host")
}

func findFirewallRules(pattern string, firewall *compute.Firewall) bool {
	if strings.Contains(firewall.Name, pattern) {
		if firewall.Direction == "INGRESS" {
			return true
		}
	}
	return false
}

func updateFirwallRule(value string, firewall *compute.Firewall) *compute.Firewall {
	// We just update the SourceRanges field and return the rule
	firewall.SourceRanges = []string{value}
	return firewall
}

func pushFirewallRule(ctx context.Context, computeService *compute.Service, project string, firewall *compute.Firewall) bool {
	// Push firewall rule to GCP
	_, err := computeService.Firewalls.Update(project, firewall.Name, firewall).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}
	return true
}

func logNotUpdating(w *tabwriter.Writer, rule *compute.Firewall, reason string) {
	fmt.Fprintf(w, "%s\t%s\n", rule.Name, reason)
}

func getExternalIP() string {
	resp, err := http.Get("http://ipv4.myexternalip.com/raw")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var b bytes.Buffer

	_, err = io.Copy(&b, resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	externalIP := strings.TrimSuffix(b.String(), "\n")
	return externalIP + "/32"
}

// UpdatedRule is a struct containing the updated firewall
// rule with the new SourceRanges and the old IP
type UpdatedRule struct {
	old      string
	firewall *compute.Firewall
}

func main() {
	flag.Parse()

	if creds != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", creds)
	}

	ctx := context.Background()

	httpClient, err := google.DefaultClient(ctx, compute.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
	}

	computeService, err := compute.New(httpClient)
	if err != nil {
		log.Fatal(err)
	}

	project := "gcp-sd-sre-p-osd"

	externalIP := getExternalIP()

	fmt.Println()
	fmt.Println("Firewall rules will be updated with the external IP: " + externalIP)

	firewallRulesToUpdate := []*UpdatedRule{}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, '\t', tabwriter.AlignRight)
	fmt.Println("")
	fmt.Fprintln(w, "Not updating\tReason")
	fmt.Fprintln(w, "------------\t------")
	req := computeService.Firewalls.List(project)
	if err := req.Pages(ctx, func(page *compute.FirewallList) error {
		for _, firewall := range page.Items {
			if findFirewallRules(pattern, firewall) {
				if len(firewall.SourceRanges) > 1 {
					logNotUpdating(w, firewall, fmt.Sprintf("More than 1 SourceRanges [%s]", strings.Join(firewall.SourceRanges, ", ")))
					continue
				}
				if len(firewall.SourceRanges) == 0 {
					if len(firewall.SourceTags) > 0 {
						logNotUpdating(w, firewall, fmt.Sprintf("Uses SourceTags [%s]", strings.Join(firewall.SourceTags, ", ")))
					} else {
						logNotUpdating(w, firewall, "No SourceRanges or SourceTags")
					}
					continue
				}
				if force || firewall.SourceRanges[0] == "0.0.0.0/0" {
					oldIP := firewall.SourceRanges[0]
					firewall = updateFirwallRule(externalIP, firewall)
					firewallRulesToUpdate = append(firewallRulesToUpdate, &UpdatedRule{oldIP, firewall})

				} else {
					logNotUpdating(w, firewall, fmt.Sprintf("Has specific IP set [%s]", strings.Join(firewall.SourceRanges, ", ")))
				}
			}
		}
		w.Flush()
		if len(firewallRulesToUpdate) == 0 {
			fmt.Println("\nThere are no rules to update! Goodbye!")
			os.Exit(0)
		}
		fmt.Println("\nGoing to update the following rules:\n")
		for _, updatedRule := range firewallRulesToUpdate {
			fmt.Printf("%v\n", updatedRule.firewall.Name)
		}
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\nAre you sure you want to update the above rules? (y/n): ")
		text, _ := reader.ReadString('\n')
		fmt.Println("")
		text = strings.TrimSpace(text)
		text = strings.ToLower(text)
		if text == "y" {
			fmt.Fprintln(w, "Updated\tIP\tOLD IP")
			fmt.Fprintln(w, "-------\t--\t------")
			for _, updatedRule := range firewallRulesToUpdate {
				pushFirewallRule(ctx, computeService, project, updatedRule.firewall)
				fmt.Fprintf(w, "%s\t%s\t%s\n", updatedRule.firewall.Name, updatedRule.firewall.SourceRanges[0], updatedRule.old)
			}
			w.Flush()
		} else {
			fmt.Println("Ok exiting!")
			os.Exit(0)
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}
}
