# Landlord

Tool to manage users and tenants in AWS Cognito

## Basic Usage

```bash
# Ensure that all commands are run with the same AWS profile
export AWS_PROFILE=example-profile

# Ensure you are logged into the command-line for AWS
aws sso login

# Sign-up a new user. Make sure you have access to the inbox of
# the email address you provide.
./landlord signup \
    --email example@apres.dev \
    --password "Not-a-real-password1!" \
    --userpool "example-user-pool-name"

# An email will be sent from no-reply@verificationemail.com (you
# may need to check your spam folder) with a signup code. To confirm
# that code.
./landlord confirm \
    --email example@apres.dev \
    --userpool "example-user-pool-name" \
    --code "123456"

# Now, you can login that user with
./landlord login \
    --email example@apres.dev \
    --password "Not-a-real-password1!" \
    --userpool "example-user-pool-name"

# You can show your session information with
./landlord info
User pool :  example-user-pool-name
Email     : matt@apres.dev
Tokens:
  Access  : valid
  ID      : valid
  Refresh : valid

# With the verbose flag, the contents of the keys will be displayed
./landlord info --verbose

# You can logout the current session (stored in `session.json`)
./landlord logout
```

## Admin Usage

```bash
# Ensure that all commands are run with the same AWS profile
export AWS_PROFILE=example-profile

# Ensure you are logged into the command-line for AWS
aws sso login

# List all users in a user pool
./landlord user list --userpool "example-user-pool-name"

# Delete a user
./landlord user delete --email example@apres.dev
```
