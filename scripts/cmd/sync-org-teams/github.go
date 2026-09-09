package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const api = "https://api.github.com"

// client is a minimal GitHub REST API client for org team operations.
type client struct {
	token string
	http  *http.Client
}

type teamInfo struct {
	Description string `json:"description"`
}

func (c *client) do(method, path string, body any, out any) (int, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		r = bytes.NewReader(b)
	}
	url := path
	if !strings.HasPrefix(path, "http") {
		url = api + path
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, string(data))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

// getTeam returns team info, or nil if the team does not exist (404).
func (c *client) getTeam(org, slug string) (*teamInfo, error) {
	var info teamInfo
	status, err := c.do(http.MethodGet, fmt.Sprintf("/orgs/%s/teams/%s", org, slug), nil, &info)
	if status == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// createTeam creates a new org team. GitHub derives the slug from name, so we
// pass the maintainers.yaml team name (already slug-shaped) as the name. Teams
// are created "closed" (visible to all org members).
func (c *client) createTeam(org, name, desc string) error {
	_, err := c.do(http.MethodPost, fmt.Sprintf("/orgs/%s/teams", org),
		map[string]string{"name": name, "description": desc, "privacy": "closed"}, nil)
	return err
}

func (c *client) setDescription(org, slug, desc string) error {
	_, err := c.do(http.MethodPatch, fmt.Sprintf("/orgs/%s/teams/%s", org, slug),
		map[string]string{"description": desc}, nil)
	return err
}

func (c *client) teamMembers(org, slug string) ([]string, error) {
	var members []string
	for page := 1; ; page++ {
		var batch []struct {
			Login string `json:"login"`
		}
		path := fmt.Sprintf("/orgs/%s/teams/%s/members?per_page=100&page=%d", org, slug, page)
		if _, err := c.do(http.MethodGet, path, nil, &batch); err != nil {
			return nil, err
		}
		for _, m := range batch {
			members = append(members, m.Login)
		}
		if len(batch) < 100 {
			break
		}
	}
	return members, nil
}

func (c *client) addMember(org, slug, login string) error {
	_, err := c.do(http.MethodPut, fmt.Sprintf("/orgs/%s/teams/%s/memberships/%s", org, slug, login),
		map[string]string{"role": "member"}, nil)
	return err
}

func (c *client) removeMember(org, slug, login string) error {
	_, err := c.do(http.MethodDelete, fmt.Sprintf("/orgs/%s/teams/%s/memberships/%s", org, slug, login), nil, nil)
	return err
}
