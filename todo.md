# TODOs

* affordance for impersonate and clear impersonate
* move terraform to apres-terraform in `apps/aws` -- a new directory that
contains terraform for apps like landlord
* BUG: styling for builtin UI for CSH
* BUG: shouldn't use admin error page when OTP fails
* React UI flow for adding MFA
* figure out terraform for how to configure email and SMS
* why do some apps ask for two codes when signing up to MFA?
* builtin login dialog should say "login again" after entering in OTP
* in the URL we send for signing up a new account, make the URL navigate to a
spot in the UI that forces a logout and then redirects back to the login page.
* invitation email could look better
* handle the "remember me" flag on login
* encode username as base64 in APIs when used in the URL
* add ability to give a cross-tenant impersonation permission
* how to handle abuse of the unauthenticated landlord APIs?
    * what does the WAF do to help here?
* apply handling
    * handle resolving cognito users that are not in dynamo
    * remove a user from their old group in apply if applicable
    * remove a group from a tenant if it no longer exists
    * adapt an existing userpool and reconcile users
* email domain guard rails
* Make it possible to show a token's contents when it has expired in the
  `info` command
* add IP address and device-fingerprinting data to signups:
  https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pool-settings-adaptive-authentication.html#user-pool-settings-adaptive-authentication-device-fingerprint
* UI
  * grab details from the user pool and user pool client to show in UI
  * friendly timestamp in frontend -- normalize to browser time
  * many htmx routes that are just HTML can be unauthorized routes #perf
  * sort jwt claims in the profile view, they are a map, so in display
    it's worth showing them in alphabetical order instead of random
  * add affordance to display a token in the UI given its base64 form
  * general styling improvements for the htmx version
    * add password fields that have the little eyeball to reveal the password
  * front end validation that the passwords match in new-password page
  * shake out only the icons we need from font-awesome
