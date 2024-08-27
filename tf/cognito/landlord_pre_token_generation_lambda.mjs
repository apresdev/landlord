// This handler runs prior to token generation and tells Cognito to
// copy some of the attributes from the ID token into the access token.
export const handler = function(event, context) {
    const userAttributes = event.request.userAttributes;
    event.response = {
        "claimsAndScopeOverrideDetails": {
            "accessTokenGeneration": {
                "claimsToAddOrOverride": {
                    'email': userAttributes['email'],
                    'custom:tenant': userAttributes['custom:tenant'],
                    'custom:impersonate_group': userAttributes['custom:impersonate_group'],
                    'custom:impersonate_user': userAttributes['custom:impersonate_user'],
                },
            }
        }
    };
    context.done(null, event);
};
