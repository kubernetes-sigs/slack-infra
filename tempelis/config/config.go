package config

type Config struct {
	Users           map[string]string `json:"users"`
	Channels        []Channel         `json:"channels"`
	Usergroups      []Usergroup       `json:"usergroups"`
	ChannelTemplate ChannelTemplate   `json:"channel_template,omitempty"`
}

type Channel struct {
	Name       string   `json:"name"`
	ID         string   `json:"id,omitempty"`
	Archived   bool     `json:"bool,omitempty"`
	Moderators []string `json:"moderators,omitempty"`
}

type Usergroup struct {
	Name        string   `json:"name,omitempty"`
	LongName    string   `json:"long_name,omitempty"`
	Members     []string `json:"members,omitempty"`
	Channels    []string `json:"channels,omitempty"`
	Description string   `json:"description,omitempty"`
	External    bool     `json:"external,omitempty"`
}

type ChannelTemplate struct {
	Pins    []string `json:"pins,omitempty"`
	Topic   string   `json:"topic,omitempty"`
	Purpose string   `json:"purpose,omitempty"`
}
