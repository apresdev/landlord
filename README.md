# Landlord

The `landlord` utility assists in managing users and tenants in AWS Cognito.
It provides a REST API for managing tenants, users, and roles. It also
provides a built-in command-line and web user interfaces for managing tenants,
users, and roles.

## Installation

The `landlord` tool is written in Go. To install it:

```bash
go install github.com/apres/landlord@latest
```

## Configuration and Usage

### AWS Cognito

Landlord configures one or more AWS Cognito user pools, and enables advanced
security on these pools. The AWS advanced security features are recommended,
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
  name or ID of the default user pool client. Tenants can have custom user
  pool clients.
* `LANDLORD_CERT_PATH=<path>`: Path to the certificate when running landlord
  in server mode with TLS.
* `LANDLORD_PRIVATE_KEY_PATH=<path>`: Path to the private key when running
  landlord in server mode with TLS.
* `LANDLORD_LOCALSTACK_ENDPOINT=<endpoint>`: If set, the landlord tool will
  use this endpoint for AWS API calls using localstack. AWS_PROFILE should
  be set to a profile that uses the localstack endpoint.

### Users and Roles

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

## Development Notes

### Using JWTs Customized by Landlord

JSON Web Tokens ([JWTs](https://jwt.io/)) are the means of communication of
identity data when using AWS Cognito and Landlord. Claims on a JWT are
attributes that are signed and can be trusted by the recipient of the JWT.

The AWS Cognito claims of interest are:

    * `cognito:groups`
    * `email`
    * `sub`

Landlord adds custom claims to the JWTs issued by AWS Cognito to support
tenancy and impersonation:

    * `custom:tenant`
    * `custom:impersonate_tenant`
    * `custom:impersonate_group`
    * `custom:impersonate_email`
    * `custom:impersonate_sub`

#### AWS Cognito Claims

The `cognito:groups` claim is used to model the roles of a user.
For example, for the role "admin" in the tenant "example-tenant", where
the tenant has the UUID: `123e4567-e89b-12d3-a456-426614174000`, the group
claim is named `tenant-123e4567-e89b-12d3-a456-426614174000-admin`. Landlord
ensures that at any given point, a user has at most one role and thus only
one entry in the `cognito:groups` claim.

The `email` claim on the JWT is a unique identifier of the user. The
uniqueness of an email address across AWS User Pools within an AWS account is
not guaranteed. However, within a set of tenants managed by Landlord,
uniqueness of the email address is enforced.

The `sub` claim is a UUID generated by Cognito to uniquely identify the
user (a.k.a. the _subject_).

#### Landlord Claims

Landlord adds two features to the JWTs issued by AWS Cognito: tenancy and
impersonation.

Tenancy is modeled by the `custom:tenant` claim. This claim is a UUID that
identifies the tenant. All JWTs issued in a Landlord managed system will
include this claim. There is always, at least, a `root` tenant defined.

Impersonation is a feature that allows a user to act as another user. When
impersonating, additional claims are added to the JWT:

  * `custom:impersonate_tenant`: The UUID of a tenant that is currently
    being impersonated.
  * `custom:impersonate_group`: The name of a group that is currently
    being impersonated.
  * `custom:impersonate_email` (optional): The `email` claim of the user
    that is currently being impersonated.
  * `custom:impersonate_sub` (optional): The `sub` claim of the user that is
    currently being impersonated.

When impersonating, the `custom:impersonate_email` and `custom:impersonate_sub`
claims are optional. If the `custom:impersonate_group` claim is set, but the
`custom:impersonate_email` and `custom:impersonate_sub` claims are not set,
the user represented by the `sub` and `email` claims should be considered part 
of the impersonated tenant and group.

There are four components of a user: `tenant`, `role`, `email`, and `sub`.
In an application consuming a Landlord JWT, to establish the four
components of a user:

  * `tenant`:
    * Is `custom:impersonate_group` set? If so, extract the tenant ID from
      that.
    * Otherwise, use the `custom:tenant` claim
  * `role`:
    * Is `custom:impersonate_group` set? If so, use that role.
    * Otherwise, use the first role in the `cognito:groups` claim.
  * `email`:
    * Is `custom:impersonate_email` set? If so, use that.
    * Otherwise, the `email` claim
    * Note that when impersonating a user, whenever an application
      identifies a user (e.g., in UI, messages or logs), it should identify
      both the original user and the impersonated user.
  * `sub`:
    * Is `custom:impersonate_sub` set? If so, use that.
    * Otherwise, the `sub` claim
    * If the `custom:impersonate_sub` claim is set, the
      `custom:impersonate_email` claim is also set. A JWT that includes one
      of these claims and not the other should be considered an invalid
      Landlord JWT.

### Testing

The `landlord` tool calls AWS APIs liberally. To that end,
we test with [localstack](https://www.localstack.cloud/).
A health check of localstack
is made prior to running unit tests that rely on localstack.
If that health check fails, those tests are skipped.

### On the Commitment of Auto-Generated Sources

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
