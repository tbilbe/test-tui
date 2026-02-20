# Backend Architecture Summary

## Hybrid Approach: API + DynamoDB

### Read Operations (via API)
- `GET /game-weeks` - List all gameweeks
- `GET /game-week/{gameWeekId}/fixtures` - Get fixtures for a specific gameweek

### Write Operations (via DynamoDB)
- Direct updates to `{PREFIX}-GameWeekFixtures` table
- No API endpoints exist for fixture manipulation (handled by streams/step functions in production)

## Authentication
- Cognito username/password → ID token for API calls
- AWS credentials (SSO/env vars) for DynamoDB access

## Environment Variables
- `API_ENDPOINT` - e.g., `https://se7-{prefix}.dev.api.playtheseven.com`
- `USER_POOL_ID` - e.g., `eu-west-2_uqwEOLO5d`
- `CLIENT_ID` - Cognito app client ID
- `PREFIX` - Environment prefix (e.g., `int-dev`)
- `AWS_REGION` - Default: `eu-west-2`
