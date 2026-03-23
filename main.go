package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/killaragorn/aicli-switch/internal/profile"
	"github.com/killaragorn/aicli-switch/internal/switcher"
	"github.com/killaragorn/aicli-switch/internal/token"
	"github.com/killaragorn/aicli-switch/internal/updater"
)

const version = "0.3.1"

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

	// Collect plain-text column values to compute widths
	type row struct {
		marker       string // "▶" or ""
		name         string
		typ          string
		subscription string
		status       string
		expiry       string
		isActive     bool
		// color helpers
		statusColor string
		subColor    string
		expiryColor string
	}

	rows := make([]row, 0, len(profiles))
	for _, p := range profiles {
		r := row{
			name:     p.Name,
			typ:      p.Type,
			isActive: p.IsActive,
		}
		if p.IsActive {
			r.marker = "▶"
		}

		switch p.Type {
		case "oauth":
			if p.TokenExpiry.IsZero() {
				r.status = "unknown"
				r.statusColor = yellow
			} else if p.IsExpired {
				r.status = "expired"
				r.statusColor = red
			} else {
				r.status = "valid"
				r.statusColor = green
			}
			if !p.TokenExpiry.IsZero() {
				remaining := time.Until(p.TokenExpiry)
				if remaining > 0 {
					r.expiry = formatDuration(remaining)
				} else {
					r.expiry = "expired"
					r.expiryColor = red
				}
			}
			if p.Subscription != "" {
				r.subscription = p.Subscription
				r.subColor = cyan
			} else {
				r.subscription = "-"
				r.subColor = dim
			}
		case "apikey":
			r.status = "ready"
			r.statusColor = green
			r.expiry = "n/a"
			r.expiryColor = dim
			r.subscription = "api key"
			r.subColor = dim
		}

		rows = append(rows, r)
	}

	// Compute max visible width for each column
	headers := [5]string{"NAME", "TYPE", "SUBSCRIPTION", "STATUS", "EXPIRY"}
	widths := [5]int{len(headers[0]), len(headers[1]), len(headers[2]), len(headers[3]), len(headers[4])}
	for _, r := range rows {
		vals := [5]int{len(r.name), len(r.typ), len(r.subscription), len(r.status), len(r.expiry)}
		for i, v := range vals {
			if v > widths[i] {
				widths[i] = v
			}
		}
	}

	gap := 3 // spacing between columns

	// Print header
	fmt.Printf("%s  %-*s  %-*s  %-*s  %-*s  %s%s\n", dim,
		widths[0], headers[0],
		widths[1], headers[1],
		widths[2], headers[2],
		widths[3], headers[3],
		headers[4], reset)

	// Print rows
	for _, r := range rows {
		// Marker column (2 chars wide)
		if r.isActive {
			fmt.Printf("%s▶%s ", green, reset)
		} else {
			fmt.Print("  ")
		}

		// Name
		fmt.Printf("%-*s", widths[0]+gap, r.name)

		// Type
		fmt.Printf("%-*s", widths[1]+gap, r.typ)

		// Subscription (with color)
		sub := r.subscription
		if r.subColor != "" {
			sub = r.subColor + r.subscription + reset
		}
		fmt.Printf("%s%-*s", sub, widths[2]+gap-len(r.subscription), "")

		// Status (with color)
		st := r.status
		if r.statusColor != "" {
			st = r.statusColor + r.status + reset
		}
		fmt.Printf("%s%-*s", st, widths[3]+gap-len(r.status), "")

		// Expiry (with color)
		exp := r.expiry
		if r.expiryColor != "" {
			exp = r.expiryColor + r.expiry + reset
		}
		fmt.Println(exp)
	}

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
	fmt.Printf(`%saicli-switch%s — Claude Code credential switcher (v%s)

  Seamlessly switch between multiple Claude Code accounts (OAuth or API Key)
  without re-authenticating. Useful when one account hits its usage limit.

%sCommands:%s
  aicli-switch add <name> [--type oauth|apikey]   Save current credentials as a named profile
                                                   (default type: oauth)
  aicli-switch rm <name>                          Delete a saved profile
  aicli-switch ls                                 List all profiles with status and expiry
  aicli-switch <name>                             Switch to a profile (cleans up auth conflicts)
  aicli-switch status                             Show active profile and credential sync status
  aicli-switch refresh [name]                     Refresh an OAuth token (default: active profile)
  aicli-switch version                            Show version
  aicli-switch help                               Show this help

%sExamples:%s
  claude login                              First, log in to a Claude account
  aicli-switch add work                     Save the current OAuth session as "work"
  claude login                              Log in to another account
  aicli-switch add personal                 Save it as "personal"
  aicli-switch work                         Switch back to "work" instantly
  aicli-switch add relay --type apikey      Add an API key profile (prompts for key)
  aicli-switch ls                           List all profiles with subscription and expiry

%sHow it works:%s
  OAuth profiles are read from ~/.claude/.credentials.json (claudeAiOauth).
  Switching cleans up conflicting auth state (API keys, OAuth tokens, cached
  account info in ~/.claude.json) so Claude Code starts fresh with the new profile.
`, bold, reset, version, bold, reset, bold, reset, bold, reset)
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
