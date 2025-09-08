# slack-inactive-detector

slack-inactive-detector detects inactive users in a Slack workspace or specific channels.

## Usage

The tool can be run as a standalone command-line application or as a container.

### Command Line Options

- `--config-path`: Path to a file containing the slack config (default: "config.json")
- `--channel`: Specific channel ID to check for inactive users (optional, checks entire workspace if not provided)
- `--dry-run`: Only report inactive users, don't take any action (default: true)

### Environment Variables

- `INACTIVE_YEARS`: Number of years to consider as the inactivity threshold (default: 1, accepts values like 1 or 2)

### Example Usage

```bash
# Check for users inactive for 1 year (default)
./slack-inactive-detector --config-path=config.json

# Check for users inactive for 2 years
INACTIVE_YEARS=2 ./slack-inactive-detector --config-path=config.json

# Check a specific channel
./slack-inactive-detector --config-path=config.json --channel=C1234567890

# Actually take action (not just dry run)
./slack-inactive-detector --config-path=config.json --dry-run=false
```

### Configuration

The tool requires a Slack configuration file (JSON format) with your Slack app credentials:

```json
{
  "accessToken": "xoxb-your-bot-token-here",
  "signingSecret": "your-signing-secret-here"
}
```

#### Setting up Slack App

1. Create a new Slack app at https://api.slack.com/apps
2. Add the following OAuth scopes to your bot token:
   - `users:read` - to list users
   - `users:read.email` - to get user email addresses (optional)
   - `channels:read` - to list channels (if checking specific channels)
   - `groups:read` - to list private channels (if checking specific channels)
3. Install the app to your workspace
4. Copy the Bot User OAuth Token to your config file

### Docker Usage

```bash
# Build the container
docker build -t slack-inactive-detector .

# Run with config mounted
docker run -v /path/to/config.json:/root/config.json slack-inactive-detector
```

### Output

The tool will output:
- Total number of users checked
- List of inactive users with their last activity date (if available)
- Summary of actions taken (if not in dry-run mode)

### Limitations

- The tool currently relies on Slack's `users.getPresence` API which may not provide complete activity information
- For more accurate activity detection, you may need to implement additional checks against conversation history
- The tool currently only reports inactive users - actual remediation actions need to be implemented based on your needs

### Security Notes

- The tool requires broad user read permissions
- Always test with `--dry-run=true` first
- Consider the privacy implications of tracking user activity
- Ensure compliance with your organization's policies before use
