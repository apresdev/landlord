package landlord.policy

import rego.v1

default allow := false

# Admins in the root tenant are allows to perform all actions
# across all tenants.
allow if {
	input.subject.user.role == "admin"
	input.subject.tenant.name == "root"
}

# Admins in other tenants are allowed to perform all actions
# within their tenant.
allow if {
    input.subject.user.role == "admin"
    input.subject.tenant.id == input.resource.tenantOwner.id
}

# All users are allowed to remove their own impersonations.
allow if {
    input.action == "delete"
    input.resource.kind == "impersonation"
    input.subject.user.id == input.resource.userOwner.id
}
