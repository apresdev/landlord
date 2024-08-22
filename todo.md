# TODOs

* resend an invitation email
* terraform for the lambda
* install instructions that indicate turning on advanced security
* add ability to give a cross-tenant impersonation permission
* replace terraform for initializing the user pool with the landlord tool
  itself -- replace `landord tenant init` with just `landlord init`
* how to handle abuse of the unauthenticated landlord APIs? 
    * what does the WAF do to help here?
* apply handling
    * remove a user from their old group in apply if applicable
    * remove a group from a tenant if it no longer exists
* REST API for refresh token handling
* list pending invitations
* email domain guard rails
* tenant role add-template (or some such) that adds a set of roles from a 
  predefined list, like "admin, user, guest" or "admin, poweruser, user, guest"
* Make it possible to show a token's contents when it has expired in the 
  `info` command
* invitation email could look better
* last error handling 
  * remove the last error cookie and put it in the session
  * add points where the error is cleared
* change the userpool and reconcile users in the user pool
* general styling improvements for the htmx version
  * add password fields that have the little eyeball to reveal the password
    * maybe save this for the React version only?
  * front end validation that the passwords match in new-password page
* Add IP address and device-fingerprinting data to signups:
  https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pool-settings-adaptive-authentication.html#user-pool-settings-adaptive-authentication-device-fingerprint
