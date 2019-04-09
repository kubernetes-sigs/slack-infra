package config

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar"
	"sigs.k8s.io/yaml"
)

type Parser struct {
	Config Config
}

func (p *Parser) Parse(reader io.Reader) error {
	var c Config
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read bytes: %v", err)
	}
	if err := yaml.UnmarshalStrict(content, &c); err != nil {
		return fmt.Errorf("failed to parse yaml: %v", err)
	}

	if err := mergeUsers(p.Config.Users, c.Users); err != nil {
		return fmt.Errorf("couldn't merge users: %v", err)
	}

	channels, err := mergeChannels(p.Config.Channels, c.Channels)
	if err != nil {
		return fmt.Errorf("couldn't merge channels: %v", err)
	}
	c.Channels = channels

	usergroups, err := mergeUsergroups(p.Config.Usergroups, c.Usergroups)
	if err != nil {
		return fmt.Errorf("couldn't merge usergroups: %v", err)
	}
	c.Usergroups = usergroups

	if !isTemplateEmpty(c.ChannelTemplate) {
		if !isTemplateEmpty(p.Config.ChannelTemplate) {
			return errors.New("can't overwrite existing channel template")
		}
		p.Config.ChannelTemplate = c.ChannelTemplate
	}

	return nil
}

func (p *Parser) ParseFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s: %v", path, err)
	}
	defer f.Close()
	if err := p.Parse(f); err != nil {
		return fmt.Errorf("failed to parse %s: %v", path, err)
	}
	return nil
}

func ParseFile(path string) (Config, error) {
	p := Parser{}
	if err := p.ParseFile(path); err != nil {
		return Config{}, err
	}
	return p.Config, nil
}

func ParseDir(path string) (Config, error) {
	p := Parser{}

	matches, err := doublestar.Glob(filepath.Join(path, "**.yaml"))
	if err != nil {
		return Config{}, fmt.Errorf("failed to find config files: %v", err)
	}

	for _, f := range matches {
		if err := p.ParseFile(f); err != nil {
			return Config{}, err
		}
	}
	return p.Config, nil
}

func mergeUsers(target map[string]string, source map[string]string) error {
	for k, v := range source {
		if _, ok := target[k]; ok {
			return fmt.Errorf("cannot overwrite users (duplicate user %s)", k)
		}
		target[k] = v
	}
	return nil
}

func mergeChannels(a []Channel, b []Channel) ([]Channel, error) {
	names := map[string]struct{}{}
	ids := map[string]struct{}{}
	for _, v := range a {
		names[v.Name] = struct{}{}
		if v.ID != "" {
			ids[v.ID] = struct{}{}
		}
	}
	for _, v := range b {
		if _, ok := names[v.Name]; ok {
			return nil, fmt.Errorf("cannot overwrite channel definitions (duplicate channel name %s)", v.Name)
		}
		if _, ok := ids[v.ID]; ok {
			return nil, fmt.Errorf("cannot overwrite channel definitions (duplicate channel ID %s)", v.Name)
		}
	}

	return append(a, b...), nil
}

func mergeUsergroups(a []Usergroup, b []Usergroup) ([]Usergroup, error) {
	names := map[string]struct{}{}
	for _, v := range a {
		names[v.Name] = struct{}{}
	}
	for _, v := range b {
		if _, ok := names[v.Name]; ok {
			return nil, fmt.Errorf("cannot usergroups (duplicate usergroup %s)", v.Name)
		}
	}

	return append(a, b...), nil
}

func isTemplateEmpty(t ChannelTemplate) bool {
	return len(t.Pins) == 0 && t.Purpose == "" && t.Topic == ""
}
