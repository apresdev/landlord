# Landlord

The `landlord` utility assists in managing users and tenants in AWS Cognito.
It provides a REST API for managing tenants, users, and roles. It also
provides a built-in command-line and web user interfaces for managing tenants,
users, and roles.

See [HELP.md](HELP.md) for end-user documentation.

## Deployment

The `landlord` tool is designed to be deployed as a Docker container. It
provides a REST API

## CI/CD Configuration

The GitHub Actions workflow builds and pushes a Docker image to Amazon ECR.
It requires the following repository-level configuration, set under
**Settings → Secrets and variables → Actions** in the GitHub repository.

### Repository Secret

| Name | Description |
|------|-------------|
| `AWS_ROLE_TO_ASSUME` | ARN of the IAM role the workflow assumes via OIDC to push to ECR (e.g. `arn:aws:iam::123456789012:role/MyGitHubActionsRole`) |

### Repository Variables

| Name | Description | Example |
|------|-------------|---------|
| `AWS_REGION` | AWS region where the ECR registry lives | `us-east-2` |
| `ECR_REPOSITORY` | Name of the ECR repository to push images to | `landlord` |

If any of these are missing the workflow will fail immediately with a message
listing the missing values.

#### Setting a repository secret

```
GitHub repo → Settings → Secrets and variables → Actions → New repository secret
```

#### Setting a repository variable

```
GitHub repo → Settings → Secrets and variables → Actions → Variables → New repository variable
```

## Configuration and Usage

### AWS Cognito

Landlord relies on an AWS Cognito user pool that is typically created by the
Apres Foundations terraform module. This module will enable advanced security.
The AWS advanced security features are recommended,
but do come with an additional cost. See:
[AWS Cognito Advanced Security](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pool-settings-advanced-security.html)

### Basic Usage

```bash
# Ensure that all commands are run with the same AWS profile
export AWS_PROFILE=example-profile

# Ensure you are logged into the command-line for AWS, for example:
aws sso login
```

### Admin Usage

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

### Environment Variables

* `LANDLORD_DB_TABLE_NAME`: The name of the DynamoDB table to use for storing
  tenant and user information. If not set, the table name defaults to
  `landlord-db`.
* `LANDLORD_ROUTES`:
    * `rest` - only register the REST/JSON endpoints
    * `webapp` - only register the HTML/Web app endpoints
    * `both` (default) - register both `rest` and `webapp`
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
* `LANDLORD_APP_NAME=<name>`: Name of the target application that Landlord
  is managing auth for. Used in email notifications and UI components.
* `LANDLORD_APP_ADMIN_EMAIL=<email>`: Email address of the application
  administrator. Used in email notifications.
* `LANDLORD_CONSOLE_URL=<name>`: URL of the Landlord Console webapp.
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
  name or ID of the default user pool client. Required only when a user
  pool has more than one app client (e.g. a separate client for an MCP server);
  a pool with a single client uses that client regardless. Tenants can have
  custom user pools and clients.
* `LANDLORD_CERT_PATH=<path>`: Path to the certificate when running landlord
  in server mode with TLS.
* `LANDLORD_PRIVATE_KEY_PATH=<path>`: Path to the private key when running
  landlord in server mode with TLS.
* `LANDLORD_LOCALSTACK_ENDPOINT=<endpoint>`: If set, the landlord tool will
  use this endpoint for AWS API calls using localstack. AWS_PROFILE should
  be set to a profile that uses the localstack endpoint.
* `LANDLORD_TERRAFORM_MODULE_VERSION=<version>`: The version of the terraform
  module that the container is being deployed with.
* `LANDLORD_SECURE_COOKIE_KEY`: Key used to sign and encrypt cookies. A
  base64 encoded string of 64 bytes.

### Users and Roles

The Landlord tool supports the creation of custom roles. By default, the
built-in `guest` role is applied to all users. Permissions attached to a
role are an app-specific responsibility. Landlord manages the
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

## Development Notes

### Testing

The `landlord` tool calls AWS APIs liberally (Cognito, DynamoDB, Lambda). Unit
tests use mocks; integration-style tests run against a local AWS stand-in and
skip themselves if none is reachable.

The preferred stand-in is the
[AWS Emulator](https://aws.amazon.com/products/developer-tools/emulator/), which
needs no AWS account or license:

```shell
scripts/aws-emulator start
eval "$(scripts/aws-emulator env)"
go test ./...
scripts/aws-emulator stop
```

The emulator generates new credentials and a new CA on each start, so run `env`
again after restarting. Any endpoint set in `AWS_ENDPOINT_URL` is treated as a
local stand-in. LocalStack (`AWS_PROFILE=localstack`) still works for older
setups, but current LocalStack releases require an auth token and the free
tier does not include Cognito user pools (`cognito-idp`).

### On the Commitment of Auto-Generated Sources

The `landlord` tool uses a few types of generated sources:

* openapi-generator that creates server stubs (`internal/rest/model_*` and
  `internal/rest/routers.go`) and a client package (`pkg/client`).
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

### DynamoDB

The DynamoDB table has the following attributes:

  * PK - String - Partition Key
  * SK - String - Sort Key
  * AltNameIndex - String - A global secondary index used to lookup entities
    by their alternate (usually human-oriented) names.

| # | PK                  | SK                  | GSI AltNameIndex      | Data        |
|---|---------------------|---------------------|-----------------------|-------------|
| 1 | `Global:*`          | `Tenant:<tenantID>` | `Tenant:<tenantName>` | Tenant JSON |
| 2 | `Tenant:<tenantID>` | `User:<userID>`     | `User:<email>`        | User JSON   |

Supported queries:

  * Get tenant by ID: GetItem like row 1
  * Get tenant by name: Query AltNameIndex like row 1
  * Get user by userID with tenant ID: GetItem like row 2
  * Get user by email without tenant ID: Query AltNameIndex GSI like row 2
  * List users for tenant: Query PK = tenant ID on 2 with "User:" prefix
  * List all tenants: Query PK = Global:* with "Tenant:" prefix

Unsupported (or, maybe "queries that require a Scan or other more expensive
operation"):

  * Get user by userID without tenantID -- no contexts currently where we
    don't have both of these values as this lookup is only done in a
    troubleshooting case when you are looking for a user in DynamoDB. The more
    common case is when a user enters an email address, and we have to determine
    the tenant (and get the user details).
  * Get all users in a role -- only use case for this is in a context where we
    check to see if there are any users in a role for a tenant before deleting
    it. A rare case, and not performance sensitive, so we query all users for a
    tenant and then filter by role in the application.
