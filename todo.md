# TODOs

* tenant user delete / tenant user add
* cache identity?
* manage sub-domains with route53?
* Generate a temporary password and email it when sending the invite
* Add auth domain and redirect host to terraform
* Groups in tokens?
* Add configuration for auth.domain app-domain and alert@domain
* Refresh token handling
* Fix command-line errors so they don't always display the help
* Cache the JWKs from Cognito
* Break down the JWT into its component in one of the list views
* Take user pool name or user pool ID
* Don't fail if a session.json is not already present
* Still show a token's contents when it has expired
* send an email via SES
* create a temporary password and email it
* add marketer roles
* for patch, PUT / SetFieldOnThing() for DELETE / ClearFieldOnThing()
* MFA
* Add IP address and device-fingerprinting data to signups
  * type UserContextDataType struct {

    // Encoded device-fingerprint details that your app collected with the Amazon
    // Cognito context data collection library. For more information, see [Adding user device and session data to API requests].
    //
    // [Adding user device and session data to API requests]: https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pool-settings-adaptive-authentication.html#user-pool-settings-adaptive-authentication-device-fingerprint
    EncodedData *string

# TODO Roles

* Admin: full access
  * Has access to all regions
  * Has access to all enterprise settings
* Marketer: has access to clearsight 360 only
  * They can toggle physician profiles
* Manager: has rep 360, has a region, has multiple territories
  * Same as admin, but only see one region
  * Do they have access to the admin dashboard?
* User: has rep 360 / dashboard, and their territory
  * No access to other stuff
* Guest

# Flow: User Signup by Invitation

1. Customer admin navigates to the Admin panel, clicks
new invitation.
  1. Enters a required email address, required tenant, and an 
  optional first and last name.
  1. POST to /api/v2/invitations to create a new invitation
    1. Can reject the invitation with a bad request if the
    tenant is not configured to allow the email domain provided.
    1. Creates a database entry for this invitation recording
    the start time.
    1. Calls the AWS Cognito SignUp API
      1. If this fails, it should mark the invitation as failed
      and return an error to the user suggesting they try again
      later or contact ClearSight support.
  1. User completes the Cognito signup flow, and is redirected
  back to the clearsight app.
  1. User "logins" to clearsight app, by being redirected
  to the base URL of the clearsight app with a newly created
  JSON Web Token.
    1. Before the base page in React is rendered the /api/v2/user/session 
    endpoint should be fetched.
    1. Middleware in the clearsight app verifies the claims on
    the JWT provided.
      1. This verification marks the invitation for this email
      address as completed and creates a User object using the
      fields from the invitation.

# Flow: Customer Admin Changes User Role

# Flow: User Asks Customer Admin to Resend the Invitation

1. User requests via out-of-band channel to get the invitation resent
  1. Admin may choose to navigate to the UI to determine if there
  is an existing outstanding invitation
    1. TODO: can landlord list pending invitations

# Flow: User Initiates Password Reset

# Flow: User Changes Email Address

