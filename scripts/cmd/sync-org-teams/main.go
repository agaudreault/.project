// Command sync-org-teams reconciles GitHub org teams from maintainers.yaml.
//
// maintainers.yaml is the authoritative source for maintainer membership and
// groups. This tool makes each managed GitHub org team's membership match the
// corresponding team in maintainers.yaml.
//
//   - Only managed teams are reconciled. A team is managed unless it sets
//     `managed: false` (e.g. emeritus, tracked but not provisioned). Pass
//     --include-unmanaged to reconcile those too.
//   - Teams that do not exist yet are created (privacy "closed").
//   - Each managed team's description is reconciled to a fixed string so the
//     GitHub UI makes clear the team is automation-managed.
//   - DRY-RUN by default: prints the plan and makes no changes. Pass --apply.
//
// Auth: set GITHUB_TOKEN to a token (GitHub App installation token or PAT) with
// org members read+write.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/argoproj/dot-project/scripts/internal/maintainers"
)

// teamDescription is applied to every managed team so it is obvious in the
// GitHub UI that membership is driven by maintainers.yaml.
const teamDescription = "Managed by automation on .project/maintainers.yaml. DO NOT EDIT manually."

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
}

func run() error {
	yamlPath := flag.String("maintainers-yaml", "", "path to maintainers.yaml (required)")
	apply := flag.Bool("apply", false, "perform writes (default: dry-run)")
	includeUnmanaged := flag.Bool("include-unmanaged", false, "also reconcile teams with managed: false")
	flag.Parse()

	if *yamlPath == "" {
		return fmt.Errorf("--maintainers-yaml is required")
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN is required")
	}

	file, err := maintainers.Load(*yamlPath)
	if err != nil {
		return err
	}
	proj, err := file.Project()
	if err != nil {
		return err
	}
	org := proj.Org
	if org == "" {
		org = "argoproj"
	}

	c := &client{token: token, http: &http.Client{Timeout: 30 * time.Second}}

	exitCode := 0
	for _, team := range proj.Teams {
		if !team.IsManaged() && !*includeUnmanaged {
			fmt.Printf("[SKIP] %s/%s is unmanaged (managed: false)\n", org, team.Name)
			continue
		}
		desired := lowerSet(team.Members)

		info, err := c.getTeam(org, team.Name)
		if err != nil {
			return err
		}

		// Create the team if it does not exist yet.
		created := info == nil
		if created {
			fmt.Printf("[CREATE TEAM] %s/%s (description %q, %d members)\n",
				org, team.Name, teamDescription, len(desired))
			if *apply {
				if err := c.createTeam(org, team.Name, teamDescription); err != nil {
					return err
				}
			} else {
				// Without --apply the team stays missing; flag it so a dry-run
				// on a PR surfaces that a create is pending.
				exitCode = 1
			}
		} else if info.Description != teamDescription {
			// Reconcile the description of an existing team.
			fmt.Printf("[DESC] %s/%s: set description -> %q\n", org, team.Name, teamDescription)
			if *apply {
				if err := c.setDescription(org, team.Name, teamDescription); err != nil {
					return err
				}
			}
		}

		// A team that was just created (or does not exist yet in a dry-run) has
		// no members; only list members for a pre-existing team.
		var current []string
		if !created {
			current, err = c.teamMembers(org, team.Name)
			if err != nil {
				return err
			}
		}
		currentSet := lowerSet(current)

		toAdd := diff(desired, currentSet)
		toRemove := diff(currentSet, desired)
		if len(toAdd) == 0 && len(toRemove) == 0 {
			fmt.Printf("[OK] %s/%s in sync (%d members)\n", org, team.Name, len(desired))
			continue
		}
		fmt.Printf("[DRIFT] %s/%s: +%v -%v\n", org, team.Name, toAdd, toRemove)
		if !*apply {
			continue
		}
		for _, login := range toAdd {
			if err := c.addMember(org, team.Name, login); err != nil {
				return err
			}
			fmt.Printf("  add %s\n", login)
		}
		for _, login := range toRemove {
			if err := c.removeMember(org, team.Name, login); err != nil {
				return err
			}
			fmt.Printf("  remove %s\n", login)
		}
	}

	if !*apply {
		fmt.Println("\n(dry-run; re-run with --apply to make changes)")
	}
	os.Exit(exitCode)
	return nil
}
