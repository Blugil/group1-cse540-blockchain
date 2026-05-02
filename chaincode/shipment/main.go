package main

// Chaincode for tracking shipments across supply chain organizations.
// Access is controlled by MSP ID; every state change is written as an event.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type ShipmentContract struct {
	contractapi.Contract
}

// Shipment stores only a SHA-256 hash of the off-chain cargo metadata (weight, volume, etc.)
// to keep the ledger lean while still allowing integrity verification via VerifyShipment.
type Shipment struct {
	DocType       string   `json:"docType"`
	ShipmentID    string   `json:"shipmentID"`
	Status        string   `json:"status"`
	Origin        string   `json:"origin"`
	Destination   string   `json:"destination"`
	CurrentHolder string   `json:"currentHolder"` // MSP ID of the org currently holding the goods
	Participants  []string `json:"participants"`  // MSP IDs allowed to read or act on this shipment
	DataHash      string   `json:"dataHash"`      // SHA-256 of off-chain cargo details
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

// ShipmentEvent is stored under a composite key (EVENT/{shipmentID}/{timestamp}) so a single
// partial-key scan retrieves the full audit trail for a shipment without scanning the whole ledger.
type ShipmentEvent struct {
	DocType   string `json:"docType"`
	EventType string `json:"eventType"`
	Actor     string `json:"actor"`
	Location  string `json:"location"`
	Timestamp string `json:"timestamp"`
	Notes     string `json:"notes"`
}

// DeviceRecord represents a registered IoT sensor bound to a shipment.
// Stored under composite key DEVICE/{shipmentID}/{deviceID}.
type DeviceRecord struct {
	DocType      string `json:"docType"`
	DeviceID     string `json:"deviceID"`
	DeviceType   string `json:"deviceType"`
	ShipmentID   string `json:"shipmentID"`
	RegisteredBy string `json:"registeredBy"`
	RegisteredAt string `json:"registeredAt"`
}

// PaginatedResult wraps a page of shipments with the CouchDB bookmark for the next page.
type PaginatedResult struct {
	Records  []*Shipment `json:"records"`
	Bookmark string      `json:"bookmark"`
}

func (s *ShipmentContract) shipmentExists(ctx contractapi.TransactionContextInterface, shipmentID string) (bool, error) {
	shipmentJSON, err := ctx.GetStub().GetState(shipmentID)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return shipmentJSON != nil, nil
}

// getClientMSPID pulls the MSP ID from the submitter's certificate rather than a function argument,
// so callers can't spoof their organization identity.
func getClientMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get client MSP ID: %v", err)
	}
	return mspID, nil
}

func isParticipant(shipment *Shipment, mspID string) bool {
	for _, p := range shipment.Participants {
		if p == mspID {
			return true
		}
	}
	return false
}

// getTxTimestamp uses the orderer's timestamp instead of time.Now() so all peers agree on the same value.
func getTxTimestamp(ctx contractapi.TransactionContextInterface) (string, error) {
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return "", fmt.Errorf("failed to get transaction timestamp: %v", err)
	}
	t := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos))
	return t.UTC().Format(time.RFC3339), nil
}

func (s *ShipmentContract) emitEvent(ctx contractapi.TransactionContextInterface, shipmentID string, event *ShipmentEvent) error {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("EVENT", []string{shipmentID, event.Timestamp})
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	err = ctx.GetStub().PutState(compositeKey, eventJSON)
	if err != nil {
		return fmt.Errorf("failed to put event state: %v", err)
	}

	err = ctx.GetStub().SetEvent(event.EventType, eventJSON)
	if err != nil {
		return fmt.Errorf("failed to set chaincode event: %v", err)
	}

	return nil
}

func (s *ShipmentContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	timestamp, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}

	sampleShipment := Shipment{
		DocType:       "shipment",
		ShipmentID:    "SHIP-001",
		Status:        "Created",
		Origin:        "Factory-Phoenix-AZ",
		Destination:   "Warehouse-Dallas-TX",
		CurrentHolder: "ManufacturerMSP",
		Participants:  []string{"ManufacturerMSP", "TransporterMSP", "WarehouseMSP", "RetailerMSP", "RecipientMSP"},
		DataHash:      computeHash("sample-weight:50kg,volume:2cbm,count:100"),
		CreatedAt:     timestamp,
		UpdatedAt:     timestamp,
	}

	shipmentJSON, err := json.Marshal(sampleShipment)
	if err != nil {
		return fmt.Errorf("failed to marshal sample shipment: %v", err)
	}

	err = ctx.GetStub().PutState(sampleShipment.ShipmentID, shipmentJSON)
	if err != nil {
		return fmt.Errorf("failed to put sample shipment: %v", err)
	}

	event := &ShipmentEvent{
		DocType:   "shipmentEvent",
		EventType: "CREATE",
		Actor:     "ManufacturerMSP",
		Location:  sampleShipment.Origin,
		Timestamp: timestamp,
		Notes:     "Ledger initialized with sample shipment SHIP-001",
	}
	return s.emitEvent(ctx, sampleShipment.ShipmentID, event)
}

// CreateShipment registers a new shipment. The caller's MSP is automatically added to
// participants if absent, so the creator always retains read access to their own shipment.
// offChainData can be any string representing cargo details — only its SHA-256 hash is stored.
func (s *ShipmentContract) CreateShipment(ctx contractapi.TransactionContextInterface, shipmentID string, origin string, destination string, participantsJSON string, offChainData string) error {
	exists, err := s.shipmentExists(ctx, shipmentID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("shipment %s already exists", shipmentID)
	}

	creatorMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	var participants []string
	err = json.Unmarshal([]byte(participantsJSON), &participants)
	if err != nil {
		return fmt.Errorf("failed to parse participants JSON: %v", err)
	}

	if !contains(participants, creatorMSP) {
		participants = append(participants, creatorMSP)
	}

	timestamp, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}

	shipment := Shipment{
		DocType:       "shipment",
		ShipmentID:    shipmentID,
		Status:        "Created",
		Origin:        origin,
		Destination:   destination,
		CurrentHolder: creatorMSP,
		Participants:  participants,
		DataHash:      computeHash(offChainData),
		CreatedAt:     timestamp,
		UpdatedAt:     timestamp,
	}

	shipmentJSON, err := json.Marshal(shipment)
	if err != nil {
		return fmt.Errorf("failed to marshal shipment: %v", err)
	}

	err = ctx.GetStub().PutState(shipmentID, shipmentJSON)
	if err != nil {
		return fmt.Errorf("failed to put shipment state: %v", err)
	}

	event := &ShipmentEvent{
		DocType:   "shipmentEvent",
		EventType: "CREATE",
		Actor:     creatorMSP,
		Location:  origin,
		Timestamp: timestamp,
		Notes:     fmt.Sprintf("Shipment %s created by %s", shipmentID, creatorMSP),
	}
	return s.emitEvent(ctx, shipmentID, event)
}

// GetShipment returns the shipment only if the caller's org is in the participant list.
func (s *ShipmentContract) GetShipment(ctx contractapi.TransactionContextInterface, shipmentID string) (*Shipment, error) {
	shipmentJSON, err := ctx.GetStub().GetState(shipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to read shipment %s: %v", shipmentID, err)
	}
	if shipmentJSON == nil {
		return nil, fmt.Errorf("shipment %s does not exist", shipmentID)
	}

	var shipment Shipment
	err = json.Unmarshal(shipmentJSON, &shipment)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal shipment: %v", err)
	}

	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return nil, err
	}
	if !isParticipant(&shipment, callerMSP) {
		return nil, fmt.Errorf("access denied: %s is not an authorized participant for shipment %s", callerMSP, shipmentID)
	}

	return &shipment, nil
}

// UpdateShipmentStatus — only the current holder can call this.
// Once a shipment reaches "Delivered" it can't be modified.
// Valid states: Created → InTransit → InWarehouse → WithRetailer → Delivered
func (s *ShipmentContract) UpdateShipmentStatus(ctx contractapi.TransactionContextInterface, shipmentID string, status string, location string, notes string) error {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return err
	}

	validStatuses := map[string]bool{
		"Created":      true,
		"InTransit":    true,
		"InWarehouse":  true,
		"WithRetailer": true,
		"Delivered":    true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status '%s'; must be one of: Created, InTransit, InWarehouse, WithRetailer, Delivered", status)
	}

	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}
	if shipment.CurrentHolder != callerMSP {
		return fmt.Errorf("access denied: only the current holder (%s) can update the shipment status", shipment.CurrentHolder)
	}

	if shipment.Status == "Delivered" {
		return fmt.Errorf("shipment %s has already been delivered and cannot be updated", shipmentID)
	}

	timestamp, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}

	shipment.Status = status
	shipment.UpdatedAt = timestamp

	shipmentJSON, err := json.Marshal(shipment)
	if err != nil {
		return fmt.Errorf("failed to marshal shipment: %v", err)
	}

	err = ctx.GetStub().PutState(shipmentID, shipmentJSON)
	if err != nil {
		return fmt.Errorf("failed to update shipment state: %v", err)
	}

	event := &ShipmentEvent{
		DocType:   "shipmentEvent",
		EventType: "STATUS_UPDATE",
		Actor:     callerMSP,
		Location:  location,
		Timestamp: timestamp,
		Notes:     fmt.Sprintf("Status changed to '%s'. %s", status, notes),
	}
	return s.emitEvent(ctx, shipmentID, event)
}

// TransferCustody hands the shipment to another org. The new holder must already be
// a registered participant — transfers to unknown parties are rejected.
func (s *ShipmentContract) TransferCustody(ctx contractapi.TransactionContextInterface, shipmentID string, newHolder string) error {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return err
	}

	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	if shipment.CurrentHolder != callerMSP {
		return fmt.Errorf("access denied: only the current holder (%s) can transfer custody", shipment.CurrentHolder)
	}

	if !isParticipant(shipment, newHolder) {
		return fmt.Errorf("new holder %s is not an authorized participant for shipment %s", newHolder, shipmentID)
	}

	if shipment.Status == "Delivered" {
		return fmt.Errorf("shipment %s has already been delivered and cannot be transferred", shipmentID)
	}

	timestamp, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}

	previousHolder := shipment.CurrentHolder
	shipment.CurrentHolder = newHolder
	shipment.Status = "InTransit"
	shipment.UpdatedAt = timestamp

	shipmentJSON, err := json.Marshal(shipment)
	if err != nil {
		return fmt.Errorf("failed to marshal shipment: %v", err)
	}

	err = ctx.GetStub().PutState(shipmentID, shipmentJSON)
	if err != nil {
		return fmt.Errorf("failed to update shipment state: %v", err)
	}

	event := &ShipmentEvent{
		DocType:   "shipmentEvent",
		EventType: "CUSTODY_TRANSFER",
		Actor:     callerMSP,
		Location:  "",
		Timestamp: timestamp,
		Notes:     fmt.Sprintf("Custody transferred from %s to %s", previousHolder, newHolder),
	}
	return s.emitEvent(ctx, shipmentID, event)
}

// VerifyShipment recomputes the SHA-256 hash and checks it against what was stored at creation.
// If there's a mismatch, the off-chain cargo data was tampered with.
func (s *ShipmentContract) VerifyShipment(ctx contractapi.TransactionContextInterface, shipmentID string, offChainData string) (bool, error) {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return false, err
	}

	expectedHash := computeHash(offChainData)
	if shipment.DataHash != expectedHash {
		return false, fmt.Errorf("data hash mismatch: on-chain=%s, computed=%s", shipment.DataHash, expectedHash)
	}

	return true, nil
}

func (s *ShipmentContract) GetShipmentHistory(ctx contractapi.TransactionContextInterface, shipmentID string) ([]*ShipmentEvent, error) {
	exists, err := s.shipmentExists(ctx, shipmentID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("shipment %s does not exist", shipmentID)
	}

	resultsIterator, err := ctx.GetStub().GetStateByPartialCompositeKey("EVENT", []string{shipmentID})
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment history: %v", err)
	}
	defer resultsIterator.Close()

	var events []*ShipmentEvent
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate history: %v", err)
		}

		var event ShipmentEvent
		err = json.Unmarshal(queryResponse.Value, &event)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal event: %v", err)
		}
		events = append(events, &event)
	}

	return events, nil
}

func (s *ShipmentContract) AuthorizeParticipant(ctx contractapi.TransactionContextInterface, shipmentID string, participant string) error {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return err
	}

	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	if shipment.CurrentHolder != callerMSP {
		return fmt.Errorf("access denied: only the current holder (%s) can authorize participants", shipment.CurrentHolder)
	}

	if isParticipant(shipment, participant) {
		return fmt.Errorf("participant %s is already authorized for shipment %s", participant, shipmentID)
	}

	timestamp, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}

	shipment.Participants = append(shipment.Participants, participant)
	shipment.UpdatedAt = timestamp

	shipmentJSON, err := json.Marshal(shipment)
	if err != nil {
		return fmt.Errorf("failed to marshal shipment: %v", err)
	}

	err = ctx.GetStub().PutState(shipmentID, shipmentJSON)
	if err != nil {
		return fmt.Errorf("failed to update shipment state: %v", err)
	}

	event := &ShipmentEvent{
		DocType:   "shipmentEvent",
		EventType: "AUTHORIZE",
		Actor:     callerMSP,
		Location:  "",
		Timestamp: timestamp,
		Notes:     fmt.Sprintf("Participant %s authorized by %s", participant, callerMSP),
	}
	return s.emitEvent(ctx, shipmentID, event)
}

// RevokeParticipant removes an org from the access list. The current holder cannot
// revoke themselves — that would leave the shipment without an accountable party.
func (s *ShipmentContract) RevokeParticipant(ctx contractapi.TransactionContextInterface, shipmentID string, participant string) error {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return err
	}

	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	if shipment.CurrentHolder != callerMSP {
		return fmt.Errorf("access denied: only the current holder (%s) can revoke participants", shipment.CurrentHolder)
	}

	if participant == callerMSP {
		return fmt.Errorf("the current holder cannot revoke their own access")
	}

	if !isParticipant(shipment, participant) {
		return fmt.Errorf("participant %s is not authorized for shipment %s", participant, shipmentID)
	}

	timestamp, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}

	var newParticipants []string
	for _, p := range shipment.Participants {
		if p != participant {
			newParticipants = append(newParticipants, p)
		}
	}
	shipment.Participants = newParticipants
	shipment.UpdatedAt = timestamp

	shipmentJSON, err := json.Marshal(shipment)
	if err != nil {
		return fmt.Errorf("failed to marshal shipment: %v", err)
	}

	err = ctx.GetStub().PutState(shipmentID, shipmentJSON)
	if err != nil {
		return fmt.Errorf("failed to update shipment state: %v", err)
	}

	event := &ShipmentEvent{
		DocType:   "shipmentEvent",
		EventType: "REVOKE",
		Actor:     callerMSP,
		Location:  "",
		Timestamp: timestamp,
		Notes:     fmt.Sprintf("Participant %s revoked by %s", participant, callerMSP),
	}
	return s.emitEvent(ctx, shipmentID, event)
}

// RegisterDevice binds an IoT sensor device to a shipment so it can post telemetry.
// Only the current holder can register devices; device type must be one of the known sensor classes.
func (s *ShipmentContract) RegisterDevice(ctx contractapi.TransactionContextInterface, shipmentID string, deviceID string, deviceType string) error {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return err
	}

	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	if shipment.CurrentHolder != callerMSP {
		return fmt.Errorf("access denied: only the current holder (%s) can register devices", shipment.CurrentHolder)
	}

	validTypes := map[string]bool{
		"temperature-sensor": true,
		"humidity-sensor":    true,
		"gps-tracker":        true,
		"pressure-sensor":    true,
		"shock-sensor":       true,
	}
	if !validTypes[deviceType] {
		return fmt.Errorf("invalid device type '%s'; must be one of: temperature-sensor, humidity-sensor, gps-tracker, pressure-sensor, shock-sensor", deviceType)
	}

	deviceKey, err := ctx.GetStub().CreateCompositeKey("DEVICE", []string{shipmentID, deviceID})
	if err != nil {
		return fmt.Errorf("failed to create device key: %v", err)
	}

	existing, err := ctx.GetStub().GetState(deviceKey)
	if err != nil {
		return fmt.Errorf("failed to check device registry: %v", err)
	}
	if existing != nil {
		return fmt.Errorf("device %s is already registered for shipment %s", deviceID, shipmentID)
	}

	timestamp, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}

	device := &DeviceRecord{
		DocType:      "device",
		DeviceID:     deviceID,
		DeviceType:   deviceType,
		ShipmentID:   shipmentID,
		RegisteredBy: callerMSP,
		RegisteredAt: timestamp,
	}

	deviceJSON, err := json.Marshal(device)
	if err != nil {
		return fmt.Errorf("failed to marshal device: %v", err)
	}

	if err = ctx.GetStub().PutState(deviceKey, deviceJSON); err != nil {
		return fmt.Errorf("failed to register device: %v", err)
	}

	event := &ShipmentEvent{
		DocType:   "shipmentEvent",
		EventType: "DEVICE_REGISTER",
		Actor:     callerMSP,
		Location:  "",
		Timestamp: timestamp,
		Notes:     fmt.Sprintf("Device %s (%s) registered by %s", deviceID, deviceType, callerMSP),
	}
	return s.emitEvent(ctx, shipmentID, event)
}

// RecordTelemetry stores an IoT sensor reading as an immutable TELEMETRY event.
// The device must have been previously registered via RegisterDevice.
// Valid sensor types: temperature, humidity, gps, pressure, shock.
func (s *ShipmentContract) RecordTelemetry(ctx contractapi.TransactionContextInterface, shipmentID string, deviceID string, sensorType string, value string, unit string, location string) error {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return err
	}

	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	if !isParticipant(shipment, callerMSP) {
		return fmt.Errorf("access denied: %s is not an authorized participant for shipment %s", callerMSP, shipmentID)
	}

	// Verify the device is registered for this shipment before accepting its reading.
	deviceKey, err := ctx.GetStub().CreateCompositeKey("DEVICE", []string{shipmentID, deviceID})
	if err != nil {
		return fmt.Errorf("failed to create device key: %v", err)
	}
	deviceJSON, err := ctx.GetStub().GetState(deviceKey)
	if err != nil {
		return fmt.Errorf("failed to check device registry: %v", err)
	}
	if deviceJSON == nil {
		return fmt.Errorf("device %s is not registered for shipment %s; call RegisterDevice first", deviceID, shipmentID)
	}

	validSensors := map[string]bool{
		"temperature": true,
		"humidity":    true,
		"gps":         true,
		"pressure":    true,
		"shock":       true,
	}
	if !validSensors[sensorType] {
		return fmt.Errorf("invalid sensor type '%s'; must be one of: temperature, humidity, gps, pressure, shock", sensorType)
	}

	timestamp, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}

	event := &ShipmentEvent{
		DocType:   "shipmentEvent",
		EventType: "TELEMETRY",
		Actor:     callerMSP,
		Location:  location,
		Timestamp: timestamp,
		Notes:     fmt.Sprintf("device=%s %s=%s%s", deviceID, sensorType, value, unit),
	}
	return s.emitEvent(ctx, shipmentID, event)
}

// RecordDocument stores an IPFS CID and SHA-256 hash on-chain, linking an off-chain
// document (certificate, manifest, inspection report) to this shipment immutably.
func (s *ShipmentContract) RecordDocument(ctx contractapi.TransactionContextInterface, shipmentID string, ipfsCID string, docHash string, docType string, description string) error {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return err
	}

	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	if !isParticipant(shipment, callerMSP) {
		return fmt.Errorf("access denied: %s is not an authorized participant for shipment %s", callerMSP, shipmentID)
	}

	timestamp, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}

	event := &ShipmentEvent{
		DocType:   "shipmentEvent",
		EventType: "DOCUMENT",
		Actor:     callerMSP,
		Location:  ipfsCID,
		Timestamp: timestamp,
		Notes:     fmt.Sprintf("type=%s hash=%s desc=%s", docType, docHash, description),
	}
	return s.emitEvent(ctx, shipmentID, event)
}

// GetShipmentsPaginated returns one page of shipments using CouchDB pagination.
// pageSize is the max records per page; bookmark is the opaque cursor from the previous call
// (pass "" for the first page). Returns records plus a bookmark for the next page.
func (s *ShipmentContract) GetShipmentsPaginated(ctx contractapi.TransactionContextInterface, pageSizeStr string, bookmark string) (*PaginatedResult, error) {
	pageSize, err := strconv.ParseInt(pageSizeStr, 10, 32)
	if err != nil || pageSize <= 0 {
		return nil, fmt.Errorf("invalid pageSize '%s': must be a positive integer", pageSizeStr)
	}

	queryString := `{"selector":{"docType":"shipment"}}`
	resultsIterator, responseMetadata, err := ctx.GetStub().GetQueryResultWithPagination(queryString, int32(pageSize), bookmark)
	if err != nil {
		return nil, fmt.Errorf("failed to execute paginated query: %v", err)
	}
	defer resultsIterator.Close()

	var shipments []*Shipment
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %v", err)
		}
		var shipment Shipment
		if err = json.Unmarshal(queryResponse.Value, &shipment); err != nil {
			continue
		}
		shipments = append(shipments, &shipment)
	}

	return &PaginatedResult{
		Records:  shipments,
		Bookmark: responseMetadata.Bookmark,
	}, nil
}

// GetAllShipments scans the full world state and returns only records with DocType "shipment",
// filtering out the event records that share the same key space.
func (s *ShipmentContract) GetAllShipments(ctx contractapi.TransactionContextInterface) ([]*Shipment, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get all shipments: %v", err)
	}
	defer resultsIterator.Close()

	var shipments []*Shipment
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate: %v", err)
		}

		var shipment Shipment
		err = json.Unmarshal(queryResponse.Value, &shipment)
		if err != nil {
			continue
		}
		if shipment.DocType == "shipment" {
			shipments = append(shipments, &shipment)
		}
	}

	return shipments, nil
}

// getShipmentInternal is used by write transactions. Unlike GetShipment it skips the
// participant check — each calling function is responsible for its own access control.
func (s *ShipmentContract) getShipmentInternal(ctx contractapi.TransactionContextInterface, shipmentID string) (*Shipment, error) {
	shipmentJSON, err := ctx.GetStub().GetState(shipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to read shipment %s: %v", shipmentID, err)
	}
	if shipmentJSON == nil {
		return nil, fmt.Errorf("shipment %s does not exist", shipmentID)
	}

	var shipment Shipment
	err = json.Unmarshal(shipmentJSON, &shipment)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal shipment: %v", err)
	}
	return &shipment, nil
}

func computeHash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// main supports two startup modes. When CORE_CHAINCODE_SERVER_ADDRESS is set, the chaincode
// runs as a standalone gRPC server (CCaaS), which avoids Docker-in-Docker on the peer.
// Without that variable it falls back to the standard embedded lifecycle.
func main() {
	chaincode, err := contractapi.NewChaincode(&ShipmentContract{})
	if err != nil {
		log.Fatalf("Error creating shipment chaincode: %v", err)
	}

	serverAddr := os.Getenv("CORE_CHAINCODE_SERVER_ADDRESS")
	if serverAddr != "" {
		server := &shim.ChaincodeServer{
			CCID:    os.Getenv("CORE_CHAINCODE_ID"),
			Address: serverAddr,
			CC:      chaincode,
			TLSProps: shim.TLSProperties{Disabled: true},
		}
		if err := server.Start(); err != nil {
			log.Fatalf("Error starting chaincode server: %v", err)
		}
	} else {
		if err := chaincode.Start(); err != nil {
			log.Fatalf("Error starting shipment chaincode: %v", err)
		}
	}
}
