# TODOs

* change API responses to send back tenant IDs instead of names
* add global secondary index to the tenant table for lookup by name and also
lookup all tenants
* select browser storage based on login kind
* grab details from the user pool and user pool client to show in UI
* cancel invitation action button in the tenant space
* shake out only the icons we need from font-awesome
* command to remove otp from user
* in the URL we send for signing up a new account, make the URL navigate to a
spot in the UI that forces a logout and then redirects back to the login page.
* handle the "remember me" flag on login
* encode username as base64 in APIs when used in the URL
* add ability to give a cross-tenant impersonation permission
* how to handle abuse of the unauthenticated landlord APIs? 
    * what does the WAF do to help here?
* apply handling
    * handle resolving cognito users that are not in dynamo
    * remove a user from their old group in apply if applicable
    * remove a group from a tenant if it no longer exists
* REST API for refresh token handling
* further limit the permissions needed by landlord
* list pending invitations
* email domain guard rails
* Make it possible to show a token's contents when it has expired in the 
  `info` command
* invitation email could look better
* last error handling 
  * remove the last error cookie and put it in the session
  * add points where the error is cleared
* adapt an existing userpool and reconcile users
* general styling improvements for the htmx version
  * add password fields that have the little eyeball to reveal the password
    * maybe save this for the React version only?
  * front end validation that the passwords match in new-password page
* add IP address and device-fingerprinting data to signups:
  https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pool-settings-adaptive-authentication.html#user-pool-settings-adaptive-authentication-device-fingerprint
