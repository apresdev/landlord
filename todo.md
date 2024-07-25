# TODOs

* replace "apply" with direct manipulation of the user's groups
* rename default-user-pool to landlord-default-user-pool
* how to handle abuse of the landlord APIs? -- especially "recover-password"
* remove a user from their old group in apply if applicable
* remove a group from a tenant if it no longer exists
* API for refresh token handling
* list pending invitations
* email domain guard rails
* tenant role add-template (or some such) that adds a set of roles from a 
  predefined list, like "admin, user, guest" or "admin, poweruser, user, guest"
* Still show a token's contents when it has expired
* Send an email via SES -- needs terraform
* invitation email could look better
* remove the last error cookie and put it in the session
* add points where the error is cleared
* change "signup" to "register"?
* general styling improvements for the htmx version
    * add password fields that have the little eyeball to reveal the password
      * maybe save this for the React version only?
* test user in multiple tenants
* front end validation that the passwords match in new-password page
* Add IP address and device-fingerprinting data to signups
  * type UserContextDataType struct {

    // Encoded device-fingerprint details that your app collected with the Amazon
    // Cognito context data collection library. For more information, see [Adding user device and session data to API requests].
    //
    // [Adding user device and session data to API requests]: https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pool-settings-adaptive-authentication.html#user-pool-settings-adaptive-authentication-device-fingerprint
    EncodedData *string
* MFA

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

# Flow: User Initiates Password Reset

# Flow: User Changes Email Address

