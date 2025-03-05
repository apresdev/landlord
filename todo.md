# TODOs

* add pk/sk as separate fields
* BUG: remove cognito ID from subject listUsers API
* React UI flow for adding MFA
* add indicators to submits https://htmx.org/attributes/hx-indicator/
* figure out terraform for how to configure email and SMS
* builtin login dialog should say "login again" after entering in OTP
* invitation email could look better (Cognito 2.0)
* how to handle abuse of the unauthenticated landlord APIs?
    * what does the WAF do to help here?
* apply handling
    * handle resolving cognito users that are not in dynamo
    * remove a user from their old group in apply if applicable
    * remove a group from a tenant if it no longer exists
    * adapt an existing userpool and reconcile users
* email domain guard rails
* add IP address and device-fingerprinting data to signups:
  https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pool-settings-adaptive-authentication.html#user-pool-settings-adaptive-authentication-device-fingerprint
* UI
  * friendly timestamp in frontend -- normalize to browser time
  * many htmx routes that are just HTML can be unauthorized routes #perf
  * add affordance to display a token in the UI given its base64 form
  * general styling improvements for the htmx version
    * add password fields that have the little eyeball to reveal the password
  * front end validation that the passwords match in new-password page
  * shake out only the icons we need from font-awesome
