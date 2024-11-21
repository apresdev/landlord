package landlord.policy

import rego.v1

default allow := false

# Admins in the root tenant are allows to perform all operations
# across all tenants.
allow if {
	input.principal.user.role == "admin"
	input.principal.tenant.name == "root"
}

# Admins in other tenants are allowed to perform all operations
# within their tenant.
allow if {
    input.principal.user.role == "admin"
    input.principal.tenant.id == input.resource.tenantOwner.id
}

# All users are allowed to remove their own impersonations.
allow if {
    input.operation == "delete"
    input.resource.kind == "impersonation"
    input.principal.user.id == input.resource.userOwner.id
}
