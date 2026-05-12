package domain

import "testing"

func TestRoleHasPermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		role               UserRole
		requiredPermission Permission
		expected           bool
	}{
		{
			name:               "passenger can create own order",
			role:               UserRolePassenger,
			requiredPermission: PermissionPassengerCreateOrder,
			expected:           true,
		},
		{
			name:               "passenger cannot manage tariffs",
			role:               UserRolePassenger,
			requiredPermission: PermissionAdminManageTariffs,
			expected:           false,
		},
		{
			name:               "driver can update location",
			role:               UserRoleDriver,
			requiredPermission: PermissionDriverUpdateLocation,
			expected:           true,
		},
		{
			name:               "dispatcher can assign driver",
			role:               UserRoleDispatcher,
			requiredPermission: PermissionDispatcherAssignDriver,
			expected:           true,
		},
		{
			name:               "admin can manage commissions",
			role:               UserRoleAdmin,
			requiredPermission: PermissionAdminManageCommissions,
			expected:           true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual := RoleHasPermission(test.role, test.requiredPermission)
			if actual != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func TestPermissionsForRoleReturnsCopy(t *testing.T) {
	t.Parallel()

	permissions := PermissionsForRole(UserRolePassenger)
	if len(permissions) == 0 {
		t.Fatal("expected passenger permissions")
	}

	permissions[0] = PermissionAdminManageTariffs

	if RoleHasPermission(UserRolePassenger, PermissionAdminManageTariffs) {
		t.Fatal("expected returned permissions slice to be a copy")
	}
}
