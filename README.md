# Landlord

The `landlord` tool is a command-line utility to manage users and tenants in
AWS Cognito.

## Installation

The `landlord` tool is written in Go. To install it, you can use the following
command:

```bash
go install github.com/apres/landlord@latest
```

## Testing

The `landlord` tool calls AWS APIs liberally. To that end,
we test with [localstack](https://www.localstack.cloud/).
A health check of localstack
is made prior to running unit tests that rely on localstack.
If that health check fails, those tests are skipped.

## Basic Usage

```bash
# Ensure that all commands are run with the same AWS profile
export AWS_PROFILE=example-profile

# Ensure you are logged into the command-line for AWS, for example:
aws sso login

# Sign-up a new user. Make sure you have access to the inbox of
# the email address you provide.
./landlord signup \
    --email example@apres.dev \
    --password "Not-a-real-password1!" \
    --userpool "example-user-pool-name"

# An email will be sent from no-reply@verificationemail.com (you
# may need to check your spam folder) with a signup code. To confirm
# that code.
./landlord confirm \
    --email example@apres.dev \
    --userpool "example-user-pool-name" \
    --code "123456"

# Now, you can login that user with
./landlord login \
    --email example@apres.dev \
    --password "Not-a-real-password1!" \
    --userpool "example-user-pool-name"

# You can show your session information with
./landlord info
User pool : example-user-pool-name
Email     : matt@apres.dev
Tokens:
  Access  : valid
  ID      : valid
  Refresh : valid

# With the verbose flag, the contents of the keys will be displayed
./landlord info --verbose

# You can logout the current session (stored in `session.json`)
./landlord logout
```

## Admin Usage

```bash
# Ensure that all commands are run with the same AWS profile
export AWS_PROFILE=example-profile

# Ensure you are logged into the command-line for AWS, for example:
aws sso login

# List all users in a user pool
./landlord user list --userpool "example-user-pool-name"

# Delete a user
./landlord user delete --email example@apres.dev
```

## Environment Variables

* `AWS_REGION=<region>`: If set, the landlord tool will use this region as the
  default region for all commands that require a region.
* `LANDLORD_APP_URL=<url>`: URL of the application. Where cookies
  from authentication are stored and where the app is expected to be
  delivered from.
* `LANDLORD_APP_NAME=<name>`: Name of the application. Used in email
  notifications and UI components.
* `LANDLORD_APP_ADMIN_EMAIL=<email>`: Email address of the application
  administrator. Used in email notifications.
* `LANDLORD_SENDER_EMAIL=<email>`: Email address to send alerts and invites
  from.
* `LANDLORD_LISTEN_ADDR=<address>`: Address to listen on when running
  landlord in server mode.
* `LANDLORD_IP_ALLOW_LIST=<comma-separated list of IPs>`: If set, the landlord
  tool will only allow requests from the given IPs. This is useful for
  restricting access to the landlord tool to a specific set of IPs.
* `LANDLORD_DEFAULT_IDENTITY_POOL=<identity-pool-name-or-id>`: The name or
  ID of the default identity pool. There is a single identity pool that all
  user pools are configured with.
* `LANDLORD_DEFAULT_USER_POOL=<user-pool-name-or-id>`: The name or ID of the
  default user pool. Tenants can have custom user pools, but the fallback is
  to use the default user pool.
* `LANDLORD_DEFAULT_USER_POOL_CLIENT=<user-pool-client-name-or-id>`: The
  name or ID of the default user pool client. Tenants can have custom user
  pool clients.
* `LANDLORD_CERT_PATH=<path>`: Path to the certificate when running landlord
  in server mode with TLS.
* `LANDLORD_PRIVATE_KEY_PATH=<path>`: Path to the private key when running
  landlord in server mode with TLS.
* `LANDLORD_LOCALSTACK_ENDPOINT=<endpoint>`: If set, the landlord tool will
  use this endpoint for AWS API calls using localstack. AWS_PROFILE should
  be set to a profile that uses the localstack endpoint.

## Users and Roles

The landlord tool supports the creation of custom roles. By default, the
built-in `guest` role is applied to all users. Permissions attached to a
role are an app-specific responsibility. The landlord merely manages the
assignment of users to roles and the inclusion of those roles in the
production of JWTs after a user logs in.

To create a new role for a tenant:

```bash
./landlord tenant role create --name "admin" --tenant "example-tenant"
```

To list the roles for a tenant:

```bash
./landlord tenant role list --tenant "example-tenant"
```

To assign a role to a user:

```bash
./landlord user role \
  --email "user1@example.com" --role "admin" --tenant "example-tenant"
```
