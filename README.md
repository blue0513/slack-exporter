# slack-extractor

A Go CLI tool to extract threads and replies from a Slack channel.

## Features

- Fetches top-level messages (thread parents) from a Slack channel
- Retrieves all replies for each thread
- Outputs messages formatted by thread

## Required Environment Variables

- `SLACK_BOT_TOKEN`: Your Slack Bot Token (e.g., `xoxb-...`)
- `SLACK_CHANNEL_ID`: The target channel's ID (e.g., `C1234567890`)

## Usage

1. Set the required environment variables:

```sh
export SLACK_BOT_TOKEN="xoxb-xxxx"
export SLACK_CHANNEL_ID="C1234567890"
```

2. Run the program:

```sh
go run main.go --fetch-limit 1
```

- `--fetch-limit`: Number of pages (100 thread parents per page) to fetch from `conversations.history` API

## Example Output

```
# Thread
https://slack.com/archives/C1234567890/p1681234567890123
## Message
Parent message text
## Message
Reply 1
## Message
Reply 2
...
```

## Notes

- If the Slack API rate limit is reached, the tool will automatically wait and retry
- The `SLACK_BOT_TOKEN` must have `conversations.history` and `conversations.replies` permissions
