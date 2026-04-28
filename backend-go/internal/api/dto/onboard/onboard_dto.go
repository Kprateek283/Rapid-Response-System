package onboard

// --- QR Scan DTOs ---

type ScanRequest struct {
	RoomNumber        string `json:"room_number"`
	QRIndex           int    `json:"qr_index"`
	DeviceFingerprint string `json:"device_fingerprint"`
}

type BLEPairingInfo struct {
	MACAddress    string `json:"mac_address"`
	HardwareColor string `json:"hardware_color"`
}

type ScanResponse struct {
	Status string   `json:"status"`
	Data   ScanData `json:"data"`
}

type ScanData struct {
	Token          string         `json:"token"`
	GuestID        string         `json:"guest_id"`
	RoomID         string         `json:"room_id"`
	BLEPairingInfo BLEPairingInfo `json:"ble_pairing_info"`
}

// --- Pair Status DTOs ---

type PairStatusRequest struct {
	Status       string `json:"status"`
	BatteryLevel int    `json:"battery_level"`
}

type PairStatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
