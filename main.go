package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/xbot/cc-switch/internal/profile"
	"github.com/xbot/cc-switch/internal/switcher"
)

const version = "0.1.0"

// ANSI colors
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	cyan    = "\033[36m"
	dim     = "\033[2m"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		cmdHelp()
		return
	}

	var err error
	switch args[0] {
	case "add":
		err = cmdAdd(args[1:])
	case "rm", "remove":
		err = cmdRemove(args[1:])
	case "ls", "list":
		err = cmdList()
	case "status":
		err = cmdStatus()
	case "refresh":
		err = cmdRefresh(args[1:])
	case "help", "--help", "-h":
		cmdHelp()
	case "version", "--version", "-v":
		fmt.Printf("cc-switch %s\n", version)
	default:
		// Treat as profile name to switch to
		err = switcher.Switch(args[0])
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s%serror:%s %v\n", bold, red, reset, err)
		os.Exit(1)
	}
}

func cmdAdd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cc-switch add <name> [--type oauth|apikey]")
	}

	name := args[0]
	profileType := "oauth" // default

	for i := 1; i < len(args); i++ {
		if (args[i] == "--type" || args[i] == "-t") && i+1 < len(args) {
			profileType = args[i+1]
			i++
		}
	}

	return profile.Add(name, profileType)
}

func cmdRemove(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cc-switch rm <name>")
	}
	name := args[0]
	if err := profile.Remove(name); err != nil {
		return err
	}
	fmt.Printf("Profile %q removed\n", name)
	return nil
}

func cmdList() error {
	profiles, err := profile.List()
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles found. Use 'cc-switch add <name>' to add one.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%s  NAME\tTYPE\tEMAIL\tSTATUS\tEXPIRY%s\n", dim, reset)

	for _, p := range profiles {
		marker := "  "
		if p.IsActive {
			marker = green + "▶ " + reset
		}

		status := ""
		expiry := ""
		switch p.Type {
		case "oauth":
			if p.TokenExpiry.IsZero() {
				status = yellow + "unknown" + reset
			} else if p.IsExpired {
				status = red + "expired" + reset
			} else {
				status = green + "valid" + reset
			}
			if !p.TokenExpiry.IsZero() {
				remaining := time.Until(p.TokenExpiry)
				if remaining > 0 {
					expiry = formatDuration(remaining)
				} else {
					expiry = red + "expired" + reset
				}
			}
		case "apikey":
			status = green + "ready" + reset
			expiry = dim + "n/a" + reset
		}

		email := p.Email
		if email == "" {
			email = dim + "-" + reset
		}

		fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\n", marker, p.Name, p.Type, email, status, expiry)
	}

	w.Flush()
	return nil
}

func cmdStatus() error {
	active := profile.GetActive()
	if active == "" {
		fmt.Println("No active profile. Use 'cc-switch <name>' to switch.")
		return nil
	}

	p, err := profile.Get(active)
	if err != nil {
		return err
	}

	fmt.Printf("%s%sActive Profile:%s %s\n", bold, cyan, reset, p.Name)
	fmt.Printf("  Type:    %s\n", p.Type)
	if p.Email != "" {
		fmt.Printf("  Email:   %s\n", p.Email)
	}
	fmt.Printf("  Created: %s\n", p.CreatedAt)
	if p.LastSwitched != "" {
		fmt.Printf("  Switched: %s\n", p.LastSwitched)
	}

	if p.Type == "oauth" {
		profiles, _ := profile.List()
		for _, info := range profiles {
			if info.Name == active {
				if !info.TokenExpiry.IsZero() {
					remaining := time.Until(info.TokenExpiry)
					if remaining > 0 {
						fmt.Printf("  Token:   %svalid%s (expires in %s)\n", green, reset, formatDuration(remaining))
					} else {
						fmt.Printf("  Token:   %sexpired%s\n", red, reset)
					}
				}
				break
			}
		}
	}

	return nil
}

func cmdRefresh(args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	} else {
		name = profile.GetActive()
	}

	if name == "" {
		return fmt.Errorf("no profile specified and no active profile")
	}

	return switcher.RefreshProfile(name)
}

func cmdHelp() {
	fmt.Printf(`%scc-switch%s — Claude Code OAuth account switcher (v%s)

%sUsage:%s
  cc-switch add <name> [--type oauth|apikey]   Add a new profile
  cc-switch rm <name>                          Remove a profile
  cc-switch ls                                 List all profiles
  cc-switch <name>                             Switch to a profile
  cc-switch status                             Show current profile
  cc-switch refresh [name]                     Refresh OAuth token
  cc-switch help                               Show this help

%sExamples:%s
  cc-switch add work                   Import current OAuth session as "work"
  cc-switch add relay --type apikey    Add an API key profile
  cc-switch work                       Switch to "work" profile
  cc-switch ls                         List all profiles with status
`, bold, reset, version, bold, reset, bold, reset)
}

func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}
