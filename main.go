package main

import (
	"fmt"
	"os"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/killaragorn/aicli-switch/internal/profile"
	"github.com/killaragorn/aicli-switch/internal/switcher"
	"github.com/killaragorn/aicli-switch/internal/token"
	"github.com/killaragorn/aicli-switch/internal/updater"
)

const version = "0.2.2"

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
	// Async update check (non-blocking)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		updater.CheckForUpdate(version)
	}()

	args := os.Args[1:]
	if len(args) == 0 {
		cmdHelp()
		wg.Wait()
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
		fmt.Printf("aicli-switch %s\n", version)
	default:
		// Treat as profile name to switch to
		err = switcher.Switch(args[0])
	}

	wg.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s%serror:%s %v\n", bold, red, reset, err)
		os.Exit(1)
	}
}

func cmdAdd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: aicli-switch add <name> [--type oauth|apikey]")
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
		return fmt.Errorf("usage: aicli-switch rm <name>")
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
		fmt.Println("No profiles found. Use 'aicli-switch add <name>' to add one.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%s  NAME\tTYPE\tSUBSCRIPTION\tSTATUS\tEXPIRY%s\n", dim, reset)

	for _, p := range profiles {
		marker := "  "
		if p.IsActive {
			marker = green + "▶ " + reset
		}

		status := ""
		expiry := ""
		subscription := ""
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
			if p.Subscription != "" {
				subscription = cyan + p.Subscription + reset
			} else {
				subscription = dim + "-" + reset
			}
		case "apikey":
			status = green + "ready" + reset
			expiry = dim + "n/a" + reset
			subscription = dim + "api key" + reset
		}

		fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\n", marker, p.Name, p.Type, subscription, status, expiry)
	}

	w.Flush()
	return nil
}

func cmdStatus() error {
	active := profile.GetActive()
	if active == "" {
		fmt.Println("No active profile. Use 'aicli-switch <name>' to switch.")
		return nil
	}

	p, err := profile.Get(active)
	if err != nil {
		return err
	}

	fmt.Printf("%s%sActive Profile:%s %s (%s)\n", bold, cyan, reset, p.Name, p.Type)
	fmt.Printf("  Created:  %s\n", p.CreatedAt)
	if p.LastSwitched != "" {
		fmt.Printf("  Switched: %s\n", p.LastSwitched)
	}

	if p.Type != "oauth" {
		return nil
	}

	// Read profile's saved oauth data
	profileOAuth, err := profile.ReadProfileOAuth(active)
	if err != nil {
		fmt.Printf("  %s⚠ Cannot read profile oauth data: %v%s\n", yellow, err, reset)
		return nil
	}

	fmt.Printf("\n%s%sProfile (saved):%s\n", bold, dim, reset)
	profileExpiry := token.GetExpiryFromData(profileOAuth)
	if !profileExpiry.IsZero() {
		remaining := time.Until(profileExpiry)
		if remaining > 0 {
			fmt.Printf("  Token:        %svalid%s (expires in %s, at %s)\n", green, reset, formatDuration(remaining), profileExpiry.Local().Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("  Token:        %sexpired%s (at %s)\n", red, reset, profileExpiry.Local().Format("2006-01-02 15:04:05"))
		}
	}
	if profileOAuth.SubscriptionType != "" {
		fmt.Printf("  Subscription: %s%s%s\n", cyan, profileOAuth.SubscriptionType, reset)
	}

	// Read live credentials and compare
	liveOAuth, err := profile.ReadCredentialsOAuth()
	fmt.Printf("\n%s%s~/.claude/.credentials.json (live):%s\n", bold, dim, reset)
	if err != nil {
		fmt.Printf("  %s⚠ Cannot read: %v%s\n", yellow, err, reset)
		return nil
	}

	liveExpiry := token.GetExpiryFromData(liveOAuth)
	if !liveExpiry.IsZero() {
		remaining := time.Until(liveExpiry)
		if remaining > 0 {
			fmt.Printf("  Token:        %svalid%s (expires in %s, at %s)\n", green, reset, formatDuration(remaining), liveExpiry.Local().Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("  Token:        %sexpired%s (at %s)\n", red, reset, liveExpiry.Local().Format("2006-01-02 15:04:05"))
		}
	}
	if liveOAuth.SubscriptionType != "" {
		fmt.Printf("  Subscription: %s%s%s\n", cyan, liveOAuth.SubscriptionType, reset)
	}

	// Sync check
	fmt.Printf("\n")
	if profileOAuth.AccessToken == liveOAuth.AccessToken {
		fmt.Printf("  %s✓ In sync%s\n", green, reset)
	} else {
		fmt.Printf("  %s⚠ Out of sync%s — profile token differs from live credentials\n", yellow, reset)
		fmt.Printf("    Run '%saicli-switch %s%s' to re-sync.\n", bold, active, reset)
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
	fmt.Printf(`%saicli-switch%s — Claude Code OAuth account switcher (v%s)

%sUsage:%s
  aicli-switch add <name> [--type oauth|apikey]   Add a new profile
  aicli-switch rm <name>                          Remove a profile
  aicli-switch ls                                 List all profiles
  aicli-switch <name>                             Switch to a profile
  aicli-switch status                             Show current profile
  aicli-switch refresh [name]                     Refresh OAuth token
  aicli-switch help                               Show this help

%sExamples:%s
  aicli-switch add work                   Import current OAuth session as "work"
  aicli-switch add relay --type apikey    Add an API key profile
  aicli-switch work                       Switch to "work" profile
  aicli-switch ls                         List all profiles with status
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
