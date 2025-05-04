package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	baseThreadURL = "https://slack.com/archives/%s/p%s"
)

type Message struct {
	Ts       string `json:"ts"`
	Text     string `json:"text"`
	ThreadTs string `json:"thread_ts,omitempty"`
}

type HistoryResponse struct {
	Messages         []Message `json:"messages"`
	HasMore          bool      `json:"has_more"`
	ResponseMetadata struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}

func getEnv(name string) string {
	env := os.Getenv(name)
	if env == "" {
		fmt.Println(fmt.Sprintf("%s is not set.", name))
		os.Exit(1)
	}
	return env
}

func getSlackToken() string {
	return getEnv("SLACK_BOT_TOKEN")
}

func getChannelID() string {
	return getEnv("SLACK_CHANNEL_ID")
}

func doRequestWithRetry(req *http.Request) (*http.Response, error) {
	for {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == 429 {
			retryAfter := resp.Header.Get("Retry-After")
			waitSec, _ := strconv.Atoi(retryAfter)
			fmt.Printf("Rate limited. Waiting %d seconds...\n", waitSec)
			time.Sleep(time.Duration(waitSec) * time.Second)
			continue
		}

		return resp, nil
	}
}

func fetchTopLevelMessages(token, channel string, fetchPages int) ([]Message, error) {
	var messages []Message
	cursor := ""
	pagesFetched := 0

	for {
		if fetchPages > 0 && pagesFetched >= fetchPages {
			break
		}

		values := url.Values{}
		values.Set("channel", channel)
		values.Set("limit", "100")
		if cursor != "" {
			values.Set("cursor", cursor)
		}

		req, _ := http.NewRequest("GET", "https://slack.com/api/conversations.history?"+values.Encode(), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := doRequestWithRetry(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var res HistoryResponse
		if err := json.Unmarshal(body, &res); err != nil {
			return nil, err
		}

		for _, msg := range res.Messages {
			if msg.ThreadTs == "" || msg.ThreadTs == msg.Ts {
				messages = append(messages, msg)
			}
		}

		if !res.HasMore || res.ResponseMetadata.NextCursor == "" {
			break
		}

		cursor = res.ResponseMetadata.NextCursor
		pagesFetched++
	}

	return messages, nil
}

func fetchThreadReplies(token, channel, threadTs string) ([]Message, error) {
	var replies []Message
	cursor := ""

	for {
		values := url.Values{}
		values.Set("channel", channel)
		values.Set("ts", threadTs)
		values.Set("limit", "100")
		if cursor != "" {
			values.Set("cursor", cursor)
		}

		req, _ := http.NewRequest("GET", "https://slack.com/api/conversations.replies?"+values.Encode(), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := doRequestWithRetry(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var res HistoryResponse
		if err := json.Unmarshal(body, &res); err != nil {
			return nil, err
		}

		replies = append(replies, res.Messages...)

		if !res.HasMore || res.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = res.ResponseMetadata.NextCursor

		time.Sleep(time.Duration(1) * time.Second)
	}

	return replies, nil
}

func formatSlackTs(ts string) string {
	return strings.ReplaceAll(ts, ".", "")
}

func main() {
	fetchLimit := flag.Int("fetch-limit", 1, "Number of pages (100 messages per page) to fetch from conversations.history")
	flag.Parse()

	slackToken := getSlackToken()
	channelID := getChannelID()

	messages, err := fetchTopLevelMessages(slackToken, channelID, *fetchLimit)
	if err != nil {
		fmt.Println("Error fetching top-level messages:", err)
		os.Exit(1)
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Ts < messages[j].Ts
	})

	for _, msg := range messages {
		replies, err := fetchThreadReplies(slackToken, channelID, msg.Ts)
		if err != nil {
			fmt.Printf("Error fetching thread %s: %v\n", msg.Ts, err)
			continue
		}

		fmt.Println("# Thread")
		fmt.Printf(baseThreadURL+"\n", channelID, formatSlackTs(msg.Ts))
		for _, r := range replies {
			fmt.Println("## Message")
			fmt.Println(r.Text)
		}
	}
}
