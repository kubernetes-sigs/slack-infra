package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sigs.k8s.io/slack-infra/slack"
	"time"
)

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("ok"))
}

func handleURLVerification(body []byte) ([]byte, error) {
	request := struct {
		Challenge string `json:"challenge"`
	}{}
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("error parsing request: %v", err)
	}
	response := map[string]string{"challenge": request.Challenge}
	return json.Marshal(response)
}
func parseFlags() options {
	o := options{}
	flag.StringVar(&o.configPath, "config-path", "config.json", "Path to a file containing the slack config")
	flag.Parse()
	return o
}

type options struct {
	configPath string
}

func main() {
	o := parseFlags()
	fmt.Println(o.configPath)
	c := slack.Config{
		SigningSecret: os.Getenv("SIGNING_SECRET"),
		WebhookURL:    os.Getenv("WEBHOOK_URL"),
		AccessToken:   os.Getenv("ACCESS_TOKEN"),
	}
	s := slack.New(c)
	publicChannels, _ := s.GetPublicChannels()
	pretty, _ := json.MarshalIndent(publicChannels, "", "\t")
	fmt.Println(string(pretty))

	http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		fmt.Printf("\nnew request %s\n", t.Format(time.RFC3339))
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
		}
		var messageEvent slack.MessageChannelsEvent
		json.Unmarshal(body, &messageEvent)
		fmt.Println(string(body))
		if messageEvent.Event.BotID != "" {
			return
		}
		req := map[string]interface{}{
			"channel": messageEvent.Event.Channel,
			"text":    "pong",
		}
		err = s.CallMethod("chat.postMessage", req, nil)
		if err != nil {
			fmt.Println(err)
		}
		req = map[string]interface{}{
			"channel":   messageEvent.Event.Channel,
			"thread_ts": messageEvent.Event.Ts,
			"text":      "pong",
		}
		err = s.CallMethod("chat.postMessage", req, nil)
		if err != nil {
			fmt.Println(err)
		}
		req = map[string]interface{}{
			"channel": messageEvent.Event.User,
			"text":    "pong",
		}
		err = s.CallMethod("chat.postMessage", req, nil)
		if err != nil {
			fmt.Println(err)
		}
		req = map[string]interface{}{
			"channel": messageEvent.Event.Channel,
			"user":    messageEvent.Event.User,
			"text":    "pong",
		}
		err = s.CallMethod("chat.postEphemeral", req, nil)
		if err != nil {
			fmt.Println(err)
		}
		verification, err := handleURLVerification(body)
		if err != nil {
		}
		w.Write(verification)
	})

	http.HandleFunc("/healthz", handleHealthz)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
