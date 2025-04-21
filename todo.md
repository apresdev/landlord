# TODOs

* add pk/sk as separate fields
* add indicators to submits https://htmx.org/attributes/hx-indicator/
* builtin login dialog should say "login again" after entering in OTP
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
  * shake out only the icons we need from font-awesome
