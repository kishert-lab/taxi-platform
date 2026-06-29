package domain

type Permission string

const (
	PermissionPassengerCreateOrder       Permission = "passenger.orders.create"
	PermissionPassengerViewCurrentOrder  Permission = "passenger.orders.current.view"
	PermissionPassengerViewOrderHistory  Permission = "passenger.orders.history.view"
	PermissionPassengerCancelOrder       Permission = "passenger.orders.cancel"
	PermissionPassengerRateTrip          Permission = "passenger.trips.rate"
	PermissionPassengerViewDriverContact Permission = "passenger.driver.contact.view"

	PermissionDriverAuthorize          Permission = "driver.auth.authorize"
	PermissionDriverSubmitVerification Permission = "driver.verification.submit"
	PermissionDriverGoOnline           Permission = "driver.status.online"
	PermissionDriverGoOffline          Permission = "driver.status.offline"
	PermissionDriverUpdateLocation     Permission = "driver.location.update"
	PermissionDriverReceiveOrders      Permission = "driver.orders.receive"
	PermissionDriverAcceptOrder        Permission = "driver.orders.accept"
	PermissionDriverRejectOrder        Permission = "driver.orders.reject"
	PermissionDriverUpdateTripStatus   Permission = "driver.trips.status.update"
	PermissionDriverCompleteOrder      Permission = "driver.orders.complete"
	PermissionDriverViewEarnings       Permission = "driver.earnings.view"
	PermissionDriverViewBalance        Permission = "driver.balance.view"
	PermissionDriverViewTransactions   Permission = "driver.transactions.view"
	PermissionDriverViewOrderHistory   Permission = "driver.orders.history.view"

	PermissionAdminManageCities      Permission = "admin.cities.manage"
	PermissionAdminManageZones       Permission = "admin.zones.manage"
	PermissionAdminManageTariffs     Permission = "admin.tariffs.manage"
	PermissionAdminManageDrivers     Permission = "admin.drivers.manage"
	PermissionAdminBlockDrivers      Permission = "admin.drivers.block"
	PermissionAdminViewOrders        Permission = "admin.orders.view"
	PermissionAdminViewStatistics    Permission = "admin.statistics.view"
	PermissionAdminAssignDriver      Permission = "admin.orders.driver.assign"
	PermissionAdminManageCommissions Permission = "admin.commissions.manage"
	PermissionAdminViewFinance       Permission = "admin.finance.view"
	PermissionAdminViewComplaints    Permission = "admin.complaints.view"
	PermissionAdminViewRatings       Permission = "admin.ratings.view"

	PermissionDispatcherCreateOrder   Permission = "dispatcher.orders.create"
	PermissionDispatcherFindPassenger Permission = "dispatcher.passengers.find"
	PermissionDispatcherAssignDriver  Permission = "dispatcher.orders.driver.assign"
	PermissionDispatcherUpdateAddress Permission = "dispatcher.orders.address.update"
	PermissionDispatcherCancelOrder   Permission = "dispatcher.orders.cancel"
	PermissionDispatcherContactDriver Permission = "dispatcher.drivers.contact"

	PermissionTaxiParkManageProfile Permission = "taxi_park.profile.manage"
	PermissionTaxiParkCreateDrivers Permission = "taxi_park.drivers.create"
	PermissionTaxiParkManageDrivers Permission = "taxi_park.drivers.manage"
	PermissionTaxiParkManageCars    Permission = "taxi_park.cars.manage"
	PermissionTaxiParkViewOrders    Permission = "taxi_park.orders.view"
	PermissionTaxiParkViewEarnings  Permission = "taxi_park.earnings.view"
	PermissionTaxiParkViewFinance   Permission = "taxi_park.finance.view"
)

var rolePermissions = map[UserRole][]Permission{
	UserRolePassenger: {
		PermissionPassengerCreateOrder,
		PermissionPassengerViewCurrentOrder,
		PermissionPassengerViewOrderHistory,
		PermissionPassengerCancelOrder,
		PermissionPassengerRateTrip,
		PermissionPassengerViewDriverContact,
	},
	UserRoleDriver: {
		PermissionPassengerCreateOrder,
		PermissionPassengerViewCurrentOrder,
		PermissionPassengerViewOrderHistory,
		PermissionPassengerCancelOrder,
		PermissionPassengerRateTrip,
		PermissionPassengerViewDriverContact,
		PermissionDriverAuthorize,
		PermissionDriverSubmitVerification,
		PermissionDriverGoOnline,
		PermissionDriverGoOffline,
		PermissionDriverUpdateLocation,
		PermissionDriverReceiveOrders,
		PermissionDriverAcceptOrder,
		PermissionDriverRejectOrder,
		PermissionDriverUpdateTripStatus,
		PermissionDriverCompleteOrder,
		PermissionDriverViewEarnings,
		PermissionDriverViewBalance,
		PermissionDriverViewTransactions,
		PermissionDriverViewOrderHistory,
	},
	UserRoleAdmin: {
		PermissionPassengerCreateOrder,
		PermissionPassengerViewCurrentOrder,
		PermissionPassengerViewOrderHistory,
		PermissionPassengerCancelOrder,
		PermissionPassengerRateTrip,
		PermissionPassengerViewDriverContact,
		PermissionAdminManageCities,
		PermissionAdminManageZones,
		PermissionAdminManageTariffs,
		PermissionAdminManageDrivers,
		PermissionAdminBlockDrivers,
		PermissionAdminViewOrders,
		PermissionAdminViewStatistics,
		PermissionAdminAssignDriver,
		PermissionAdminManageCommissions,
		PermissionAdminViewFinance,
		PermissionAdminViewComplaints,
		PermissionAdminViewRatings,
	},
	UserRoleDispatcher: {
		PermissionPassengerCreateOrder,
		PermissionPassengerViewCurrentOrder,
		PermissionPassengerViewOrderHistory,
		PermissionPassengerCancelOrder,
		PermissionPassengerRateTrip,
		PermissionPassengerViewDriverContact,
		PermissionDispatcherCreateOrder,
		PermissionDispatcherFindPassenger,
		PermissionDispatcherAssignDriver,
		PermissionDispatcherUpdateAddress,
		PermissionDispatcherCancelOrder,
		PermissionDispatcherContactDriver,
	},
	UserRoleTaxiPark: {
		PermissionPassengerCreateOrder,
		PermissionPassengerViewCurrentOrder,
		PermissionPassengerViewOrderHistory,
		PermissionPassengerCancelOrder,
		PermissionPassengerRateTrip,
		PermissionPassengerViewDriverContact,
		PermissionTaxiParkManageProfile,
		PermissionTaxiParkCreateDrivers,
		PermissionTaxiParkManageDrivers,
		PermissionTaxiParkManageCars,
		PermissionTaxiParkViewOrders,
		PermissionTaxiParkViewEarnings,
		PermissionTaxiParkViewFinance,
	},
}

func PermissionsForRole(role UserRole) []Permission {
	permissions := rolePermissions[role]
	result := make([]Permission, len(permissions))
	copy(result, permissions)

	return result
}

func RoleHasPermission(role UserRole, requiredPermission Permission) bool {
	for _, permission := range rolePermissions[role] {
		if permission == requiredPermission {
			return true
		}
	}

	return false
}
