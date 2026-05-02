// Blockchain-Based Shipment Tracking Chaincode
// This chaincode implements a supply chain provenance system on Hyperledger Fabric.
// It tracks shipments through their full lifecycle — from manufacturer to final recipient —
// recording every custody transfer as an immutable, digitally signed transaction on the ledger.
//
// @authors: Dominick Agnello, Ritish Abrol, Vatsal Patel, Shashikant Nanda, Anushree Bhure

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// ShipmentContract provides functions for managing shipments on the blockchain
type ShipmentContract struct {
	contractapi.Contract
}

// ---- Data Models ----

// Shipment represents a shipment record stored on the ledger.
// It contains metadata about the shipment, its current status, and authorized participants.
type Shipment struct {
	DocType       string   `json:"docType"`       // "shipment" — used for CouchDB rich queries
	ShipmentID    string   `json:"shipmentID"`    // Unique identifier for the shipment
	Status        string   `json:"status"`        // Current status: Created, InTransit, InWarehouse, WithRetailer, Delivered
	Origin        string   `json:"origin"`        // Origin location of the shipment
	Destination   string   `json:"destination"`   // Final destination of the shipment
	CurrentHolder string   `json:"currentHolder"` // MSP ID of the entity currently holding the shipment
	Participants  []string `json:"participants"`  // List of MSP IDs authorized to interact with this shipment
	DataHash      string   `json:"dataHash"`      // SHA-256 hash of off-chain metadata (weight, volume, etc.)
	CreatedAt     string   `json:"createdAt"`     // ISO-8601 timestamp when the shipment was created
	UpdatedAt     string   `json:"updatedAt"`     // ISO-8601 timestamp of the most recent update
}

// ShipmentEvent represents a significant event in a shipment's lifecycle.
// Events are stored as separate composite-key entries so the full history is queryable.
type ShipmentEvent struct {
	DocType   string `json:"docType"`   // "shipmentEvent"
	EventType string `json:"eventType"` // CREATE, STATUS_UPDATE, CUSTODY_TRANSFER, AUTHORIZE, REVOKE
	Actor     string `json:"actor"`     // MSP ID of the entity that triggered the event
	Location  string `json:"location"`  // Location where the event occurred
	Timestamp string `json:"timestamp"` // ISO-8601 timestamp
	Notes     string `json:"notes"`     // Free-text notes about the event
}

// ---- Helper Functions ----

// shipmentExists checks whether a shipment with the given ID already exists on the ledger
func (s *ShipmentContract) shipmentExists(ctx contractapi.TransactionContextInterface, shipmentID string) (bool, error) {
	shipmentJSON, err := ctx.GetStub().GetState(shipmentID)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return shipmentJSON != nil, nil
}

// getClientMSPID returns the MSP ID of the transaction submitter
func getClientMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get client MSP ID: %v", err)
	}
	return mspID, nil
}

// isParticipant checks whether a given MSP ID is in the shipment's participant list
func isParticipant(shipment *Shipment, mspID string) bool {
	for _, p := range shipment.Participants {
		if p == mspID {
			return true
		}
	}
	return false
}

// getTxTimestamp returns the transaction timestamp as an ISO-8601 string
func getTxTimestamp(ctx contractapi.TransactionContextInterface) (string, error) {
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return "", fmt.Errorf("failed to get transaction timestamp: %v", err)
	}
	t := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos))
	return t.UTC().Format(time.RFC3339), nil
}

// emitEvent stores a ShipmentEvent under a composite key and also sets the chaincode event
func (s *ShipmentContract) emitEvent(ctx contractapi.TransactionContextInterface, shipmentID string, event *ShipmentEvent) error {
	// Store event as a composite key entry: EVENT~shipmentID~timestamp
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

	// Set chaincode event so SDK listeners can react in real time
	err = ctx.GetStub().SetEvent(event.EventType, eventJSON)
	if err != nil {
		return fmt.Errorf("failed to set chaincode event: %v", err)
	}

	return nil
}

// ---- Smart Contract Functions ----

// InitLedger initializes the ledger with a sample shipment for demonstration/testing.
// This runs once when the chaincode is first instantiated.
func (s *ShipmentContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	timestamp, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}

	// Create a sample shipment to seed the ledger
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

	// Record the creation event
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

// CreateShipment registers a new shipment on the ledger.
// Only an exact set of entities (participants) are authorized to interact with the shipment.
//
// Parameters:
//   - shipmentID:   unique identifier for the shipment
//   - origin:       starting location
//   - destination:  final delivery location
//   - participants: JSON-encoded string array of MSP IDs allowed to interact
//   - offChainData: arbitrary metadata string whose SHA-256 hash is stored on-chain
func (s *ShipmentContract) CreateShipment(ctx contractapi.TransactionContextInterface, shipmentID string, origin string, destination string, participantsJSON string, offChainData string) error {
	// Check if shipment already exists
	exists, err := s.shipmentExists(ctx, shipmentID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("shipment %s already exists", shipmentID)
	}

	// Get the identity of the creator
	creatorMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	// Parse participants from JSON string (Fabric passes complex types as strings)
	var participants []string
	err = json.Unmarshal([]byte(participantsJSON), &participants)
	if err != nil {
		return fmt.Errorf("failed to parse participants JSON: %v", err)
	}

	// Ensure the creator is included in the participant list
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

	// Record creation event
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

// GetShipment returns the current state of a shipment by its ID.
// Access is restricted to authorized participants only.
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

	// Enforce access control — only participants can read shipment data
	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return nil, err
	}
	if !isParticipant(&shipment, callerMSP) {
		return nil, fmt.Errorf("access denied: %s is not an authorized participant for shipment %s", callerMSP, shipmentID)
	}

	return &shipment, nil
}

// UpdateShipmentStatus updates the status of a shipment and records a status event.
// Only the current holder of the shipment is allowed to update its status.
//
// Valid statuses: Created, InTransit, InWarehouse, WithRetailer, Delivered
func (s *ShipmentContract) UpdateShipmentStatus(ctx contractapi.TransactionContextInterface, shipmentID string, status string, location string, notes string) error {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return err
	}

	// Validate the new status value
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

	// Only the current holder can update the status
	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}
	if shipment.CurrentHolder != callerMSP {
		return fmt.Errorf("access denied: only the current holder (%s) can update the shipment status", shipment.CurrentHolder)
	}

	// Prevent updates to already-delivered shipments
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

// TransferCustody transfers the shipment from the current holder to a new holder.
// This is the core "transaction" — analogous to a coin transfer in cryptocurrency.
// Only the current holder can initiate a transfer, and the new holder must be an authorized participant.
func (s *ShipmentContract) TransferCustody(ctx contractapi.TransactionContextInterface, shipmentID string, newHolder string) error {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return err
	}

	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	// Only the current holder can transfer custody
	if shipment.CurrentHolder != callerMSP {
		return fmt.Errorf("access denied: only the current holder (%s) can transfer custody", shipment.CurrentHolder)
	}

	// The new holder must be an authorized participant
	if !isParticipant(shipment, newHolder) {
		return fmt.Errorf("new holder %s is not an authorized participant for shipment %s", newHolder, shipmentID)
	}

	// Cannot transfer an already-delivered shipment
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

// VerifyShipment validates the integrity of a shipment by checking that:
// 1. The shipment exists on the ledger
// 2. Its data hash matches the provided off-chain data
// Returns true if verification passes, false otherwise.
func (s *ShipmentContract) VerifyShipment(ctx contractapi.TransactionContextInterface, shipmentID string, offChainData string) (bool, error) {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return false, err
	}

	// Recompute the hash from the provided off-chain data and compare
	expectedHash := computeHash(offChainData)
	if shipment.DataHash != expectedHash {
		return false, fmt.Errorf("data hash mismatch: on-chain=%s, computed=%s", shipment.DataHash, expectedHash)
	}

	return true, nil
}

// GetShipmentHistory retrieves the entire event history for a given shipment.
// It uses composite key queries to find all EVENT entries associated with the shipment ID.
func (s *ShipmentContract) GetShipmentHistory(ctx contractapi.TransactionContextInterface, shipmentID string) ([]*ShipmentEvent, error) {
	// Verify the shipment exists
	exists, err := s.shipmentExists(ctx, shipmentID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("shipment %s does not exist", shipmentID)
	}

	// Query all composite keys with prefix EVENT~shipmentID
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

// AuthorizeParticipant adds a new participant to the shipment's authorized list.
// Only the current holder of the shipment can authorize a new participant.
func (s *ShipmentContract) AuthorizeParticipant(ctx contractapi.TransactionContextInterface, shipmentID string, participant string) error {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return err
	}

	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	// Only the current holder can authorize new participants
	if shipment.CurrentHolder != callerMSP {
		return fmt.Errorf("access denied: only the current holder (%s) can authorize participants", shipment.CurrentHolder)
	}

	// Check if already authorized
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

// RevokeParticipant removes a participant from the shipment's authorized list.
// Only the current holder can revoke participants, and the current holder cannot revoke themselves.
func (s *ShipmentContract) RevokeParticipant(ctx contractapi.TransactionContextInterface, shipmentID string, participant string) error {
	shipment, err := s.getShipmentInternal(ctx, shipmentID)
	if err != nil {
		return err
	}

	callerMSP, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	// Only the current holder can revoke participants
	if shipment.CurrentHolder != callerMSP {
		return fmt.Errorf("access denied: only the current holder (%s) can revoke participants", shipment.CurrentHolder)
	}

	// Cannot revoke yourself
	if participant == callerMSP {
		return fmt.Errorf("the current holder cannot revoke their own access")
	}

	// Check if the participant exists in the list
	if !isParticipant(shipment, participant) {
		return fmt.Errorf("participant %s is not authorized for shipment %s", participant, shipmentID)
	}

	timestamp, err := getTxTimestamp(ctx)
	if err != nil {
		return err
	}

	// Remove the participant
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

// GetAllShipments returns all shipments stored on the ledger.
// Uses a range query across all keys (for demonstration; in production use pagination).
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
			// Skip non-shipment entries (e.g., events)
			continue
		}
		if shipment.DocType == "shipment" {
			shipments = append(shipments, &shipment)
		}
	}

	return shipments, nil
}

// ---- Internal helper (does not enforce access control — used by mutation functions) ----

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

// computeHash returns the SHA-256 hex digest of the given data string
func computeHash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// contains checks if a string is present in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ---- Main ----

func main() {
	chaincode, err := contractapi.NewChaincode(&ShipmentContract{})
	if err != nil {
		log.Fatalf("Error creating shipment chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Fatalf("Error starting shipment chaincode: %v", err)
	}
}
