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

package main

import "encoding/json"

// TextObject represents a TextObject
type TextObject struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Emoji    bool   `json:"emoji,omitempty"`
	Verbatim bool   `json:"verbatim,omitempty"`
}

// View represents a view opened by view.open
type View struct {
	Type            string        `json:"type"`
	Title           TextObject    `json:"title,omitempty"`
	Blocks          []interface{} `json:"blocks"`
	Close           TextObject    `json:"close,omitempty"`
	Submit          interface{}   `json:"submit,omitempty"`
	PrivateMetadata string        `json:"private_metadata,omitempty"`
	CallbackID      string        `json:"callback_id,omitempty"`
	ClearOnClose    bool          `json:"clear_on_close,omitempty"`
	NotifyOnClose   bool          `json:"notify_on_close,omitempty"`
	ExternalID      string        `json:"external_id,omitempty"`
}

// PlainTextInputElement represents a PlainTextInputElement
type PlainTextInputElement struct {
	Type         plainTextInputType `json:"type"`
	Placeholder  TextObject         `json:"placeholder,omitempty"`
	ActionID     string             `json:"action_id"`
	InitialValue string             `json:"initial_value,omitempty"`
	Multiline    bool               `json:"multiline,omitempty"`
	MinLength    int                `json:"min_length,omitempty"`
	MaxLength    int                `json:"max_length,omitempty"`
}
type plainTextInputType string

func (plainTextInputType) MarshalJSON() ([]byte, error) {
	return json.Marshal("plain_text_input")
}

// MultiSelectChannelElement represents a MultiSelectChannelElement
type MultiSelectChannelElement struct {
	Type             multiSelectChannelElementType `json:"type"`
	Placeholder      TextObject                    `json:"placeholder"`
	ActionID         string                        `json:"action_id"`
	InitialChannels  []string                      `json:"initial_channels,omitempty"`
	Confirm          []interface{}                 `json:"confirm,omitempty"`
	MaxSelectedItems int                           `json:"max_selected_items,omitempty"`
}
type multiSelectChannelElementType string

func (multiSelectChannelElementType) MarshalJSON() ([]byte, error) {
	return json.Marshal("multi_channels_select")
}

// SectionBlock represents a SectionBlock
type SectionBlock struct {
	Type      sectionBlockType `json:"type"`
	Text      TextObject       `json:"text"`
	BlockID   string           `json:"block_id,omitempty"`
	Fields    []TextObject     `json:"fields,omitempty"`
	Accessory []interface{}    `json:"accessory,omitempty"`
}
type sectionBlockType string

func (sectionBlockType) MarshalJSON() ([]byte, error) {
	return json.Marshal("section")
}

// ActionBlock represents a ActionBlock
type ActionBlock struct {
	Type     actionBlockType `json:"type"`
	Elements []interface{}   `json:"elements"`
	BlockID  string          `json:"block_id,omitempty"`
}
type actionBlockType string

func (actionBlockType) MarshalJSON() ([]byte, error) {
	return json.Marshal("action")
}

// InputBlock represents a InputBlock
type InputBlock struct {
	Type     inputBlockType `json:"type"`
	Label    TextObject     `json:"label"`
	Element  interface{}    `json:"element"`
	BlockID  string         `json:"block_id,omitempty"`
	Hint     TextObject     `json:"hint,omitempty"`
	Optional bool           `json:"optional,omitempty"`
}
type inputBlockType string

func (inputBlockType) MarshalJSON() ([]byte, error) {
	return json.Marshal("input")
}

// DividerBlock represents a DividerBlock
type DividerBlock struct {
	Type    dividerBlockType `json:"type"`
	BlockID string           `json:"block_id,omitempty"`
}
type dividerBlockType string

func (dividerBlockType) MarshalJSON() ([]byte, error) {
	return json.Marshal("divider")
}
