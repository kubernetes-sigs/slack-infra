/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar"
	"sigs.k8s.io/yaml"
)

var (
	emptyRegexp        = regexp.MustCompile("")
	defaultRestriction = Restrictions{Path: "*", Users: true, Channels: []*regexp.Regexp{emptyRegexp}, Usergroups: []*regexp.Regexp{emptyRegexp}, Template: true}
)

type Parser struct {
	Config Config
	parsed map[string]struct{}
}

func NewParser() *Parser {
	return &Parser{
		parsed: map[string]struct{}{},
	}
}

func (p *Parser) Parse(reader io.Reader, path string) error {
	var c Config
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read bytes: %v", err)
	}
	if err := yaml.UnmarshalStrict(content, &c); err != nil {
		return fmt.Errorf("failed to parse yaml: %v", err)
	}

	if p.Config.Users == nil {
		p.Config.Users = map[string]string{}
	}

	restrictions, err := mergeRestrictions(p.Config.Restrictions, c.Restrictions)
	if err != nil {
		return fmt.Errorf("couldn't load restrictions: %v", err)
	}
	p.Config.Restrictions = restrictions

	r := resolveRestrictions(restrictions, path)

	if err := mergeUsers(p.Config.Users, c.Users, r); err != nil {
		return fmt.Errorf("couldn't merge users: %v", err)
	}

	channels, err := mergeChannels(p.Config.Channels, c.Channels, r)
	if err != nil {
		return fmt.Errorf("couldn't merge channels: %v", err)
	}
	p.Config.Channels = channels

	usergroups, err := mergeUsergroups(p.Config.Usergroups, c.Usergroups, r)
	if err != nil {
		return fmt.Errorf("couldn't merge usergroups: %v", err)
	}
	p.Config.Usergroups = usergroups

	if !isTemplateEmpty(c.ChannelTemplate) {
		if !r.Template {
			return fmt.Errorf("can't set channel template in %s", r.Path)
		}
		if !isTemplateEmpty(p.Config.ChannelTemplate) {
			return errors.New("can't overwrite existing channel template")
		}
		p.Config.ChannelTemplate = c.ChannelTemplate
	}

	return nil
}

func (p *Parser) ParseFile(path, basedir string) error {
	if _, ok := p.parsed[path]; ok {
		return nil
	}
	p.parsed[path] = struct{}{}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s: %v", path, err)
	}
	defer f.Close()
	if !strings.HasPrefix(path, basedir) {
		return fmt.Errorf("%q is not a prefix of %q", basedir, path)
	}
	if err := p.Parse(f, path[len(basedir):]); err != nil {
		return fmt.Errorf("failed to parse %s: %v", path, err)
	}
	return nil
}

func (p *Parser) ParseDir(path string) error {
	matches, err := doublestar.Glob(filepath.Join(path, "**/*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to find config files: %v", err)
	}

	for _, f := range matches {
		if err := p.ParseFile(f, path); err != nil {
			return err
		}
	}
	return nil
}

func ParseFile(path string) (Config, error) {
	p := NewParser()
	if err := p.ParseFile(path, path); err != nil {
		return Config{}, err
	}
	return p.Config, nil
}

func ParseDir(path string) (Config, error) {
	p := NewParser()
	if err := p.ParseDir(path); err != nil {
		return Config{}, err
	}
	return p.Config, nil
}

func resolveRestrictions(restrictions []Restrictions, path string) Restrictions {
	for _, r := range restrictions {
		if match, err := doublestar.Match(r.Path, path); err == nil && match {
			return r
		}
	}
	return defaultRestriction
}

func mergeRestrictions(a []Restrictions, b []Restrictions) ([]Restrictions, error) {
	if len(a) != 0 && len(b) != 0 {
		return nil, fmt.Errorf("restrictions can only be defined once")
	}
	if len(a) != 0 {
		return a, nil
	}
	ret := make([]Restrictions, 0, len(b))
	for _, r := range b {
		r.Channels = make([]*regexp.Regexp, 0, len(r.ChannelsString))
		for _, p := range r.ChannelsString {
			re, err := regexp.Compile(p)
			if err != nil {
				return nil, fmt.Errorf("failed to parse channel pattern %q for path %q: %v", p, r.Path, err)
			}
			r.Channels = append(r.Channels, re)
		}
		r.Usergroups = make([]*regexp.Regexp, 0, len(r.UsergroupsString))
		for _, p := range r.UsergroupsString {
			re, err := regexp.Compile(p)
			if err != nil {
				return nil, fmt.Errorf("failed to parse usergroup pattern %q for path %q: %v", p, r.Path, err)
			}
			r.Usergroups = append(r.Usergroups, re)
		}
		ret = append(ret, r)
	}
	return ret, nil
}

func matchesRegexList(s string, tests []*regexp.Regexp) bool {
	for _, r := range tests {
		if r.MatchString(s) {
			return true
		}
	}
	return false
}

func mergeUsers(target map[string]string, source map[string]string, r Restrictions) error {
	if !r.Users {
		return fmt.Errorf("cannot define users in %q", r.Path)
	}
	for k, v := range source {
		if _, ok := target[k]; ok {
			return fmt.Errorf("cannot overwrite users (duplicate user %s)", k)
		}
		if len(v) != 9 {
			return fmt.Errorf("%s: %q is not a valid slack user ID", k, v)
		}
		target[k] = v
	}
	return nil
}

func mergeChannels(a []Channel, b []Channel, r Restrictions) ([]Channel, error) {
	names := map[string]struct{}{}
	ids := map[string]struct{}{}
	for _, v := range a {
		names[v.Name] = struct{}{}
		if v.ID != "" {
			ids[v.ID] = struct{}{}
		}
	}
	for _, v := range b {
		if v.Name == "" {
			return nil, fmt.Errorf("channels must have names")
		}
		if !matchesRegexList(v.Name, r.Channels) {
			return nil, fmt.Errorf("cannot define channel %q in %q", v.Name, r.Path)
		}
		if _, ok := names[v.Name]; ok {
			return nil, fmt.Errorf("cannot overwrite channel definitions (duplicate channel name %s)", v.Name)
		}
		if _, ok := ids[v.ID]; ok {
			return nil, fmt.Errorf("cannot overwrite channel definitions (duplicate channel ID %s)", v.Name)
		}
	}

	return append(a, b...), nil
}

func mergeUsergroups(a []Usergroup, b []Usergroup, r Restrictions) ([]Usergroup, error) {
	names := map[string]struct{}{}
	for _, v := range a {
		names[v.Name] = struct{}{}
	}
	for _, v := range b {
		if v.Name == "" {
			return nil, fmt.Errorf("usergroups must have names")
		}
		if !matchesRegexList(v.Name, r.Usergroups) {
			return nil, fmt.Errorf("cannot define usergroup %q in %q", v.Name, r.Path)
		}
		if !v.External {
			if v.LongName == "" {
				return nil, fmt.Errorf("usergroup %s must have a long name", v.Name)
			}
			if v.Description == "" {
				return nil, fmt.Errorf("usergroup %s must have a description", v.Name)
			}
			if len(v.Members) == 0 {
				return nil, fmt.Errorf("usergroup %s must have at least one member", v.Name)
			}
		}
		if _, ok := names[v.Name]; ok {
			return nil, fmt.Errorf("cannot usergroups (duplicate usergroup %s)", v.Name)
		}
	}

	return append(a, b...), nil
}

func isTemplateEmpty(t ChannelTemplate) bool {
	return len(t.Pins) == 0 && t.Purpose == "" && t.Topic == ""
}
