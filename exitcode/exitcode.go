package exitcode

const (
	Success      = 0 // Scan succeeded, devices found.
	GeneralError = 1 // Permission error, interface not found, scan failure, etc.
	NoDevices    = 2 // Scan completed successfully but found no devices (--json and --once only).
)
