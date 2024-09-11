# Landlord

The `landlord` tool is a command-line utility to manage users and tenants in
AWS Cognito.

## Installation

The `landlord` tool is written in Go. To install it, you can use the following
command:

```bash
go install github.com/apres/landlord@latest
```

## AWS Cognito

Landlord configures one or more AWS Cognito user pools, and enables advanced
security on these pools. The AWS advanced security features are recommended,
but do come with an additional cost. See:
[AWS Cognito Advanced Security](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pool-settings-advanced-security.html)

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

* `LANDLORD_DEV_MODE`: If set, the landlord tool will run in development mode.
  This enables various development features, such as:
    * Use of the zap development logger.
    * In server mode, static files will be served from the local
      file system instead of via an embedded file system. This simplifies
      reloading things like style sheets.
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

## On the Commitment of Auto-Generated Sources

The `landlord` tool uses a few types of generated sources:

  * openapi-generator that creates server stubs (`internal/rest/model_*` and 
    `internal/rest/routers.go`) and a client package (`pkg/client`).
  * .go files associated with [templ](https://templ.guide/) templates in the 
    `internal/server` directory. The .go files here are each created from 
    one .templ file.
  * mocks for unit testing in the `internal/mocks` directory.

Our decision is to commit these sources. The argument for not including 
generated sources typically mentions the DRY principle: "Don't Repeat 
Yourself." Admittedly, the inclusion does present a potential issue if there is 
ever drift between the two. Adhering to a DRY principle would avoid such an 
issue.

All that said, we choose to include the auto-generated sources to support 
downstream users that want to include the library as a dependency or build 
the application without need for additional tooling / steps. They should only 
need to use `go build`. 

This places the burden of ensuring that the generated sources are in sync on 
the maintainers of the `landlord` tool, which we believe is a good thing.


## DynamoDB

| PK                     | SK                     | Data              | Reason                                         |
|------------------------|------------------------|-------------------|------------------------------------------------|
| `TENANT#<tenant-id>`   | `TENANT`               | Tenant JSON       | Lookup tenant by ID                            |
| `TENANT#<tenant-name>` | `TENANT`               | tenant-id         | Lookup tenant by name                          |
| `TENANT#<tenant-id>`   | `USER#<username>`      | User JSON         | Lookup user in tenant                          |
| `USER#<username>`      | `USER`                 | tenant-id         | Lookup tenant-id for user                      |
| `TENANT#<tenant-id>`   | `ROLE#<role-name>`     | list of usernames | Lookup usernames belonging to a role           |
| `TENANT#<tenant-id>`   | `EMAIL#<email-domain>` | Email domain JSON | Lookup email domain for tenant                 | 
| `EMAIL#<email-domain>` | `EMAIL`                | tenant-id         | Lookup email domain when exclusive to a tenant |
