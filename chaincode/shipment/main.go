package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// ShipmentContract provides functions for managing shipments on the blockchain
type ShipmentContract struct {
	contractapi.Contract
}

type Shipment struct {
	DocType       string   `json:"docType"`       // "shipment" — used for CouchDB rich queries
	ShipmentID    string   `json:"shipmentID"`
	Status        string   `json:"status"`
	Origin        string   `json:"origin"`
	Destination   string   `json:"destination"`
	CurrentHolder string   `json:"currentHolder"`
	Participants  []string `json:"participants"`
	DataHash      string   `json:"dataHash"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

type ShipmentEvent struct {
	DocType   string `json:"docType"`   // "shipmentEvent"
	EventType string `json:"eventType"`
	Actor     string `json:"actor"`
	Location  string `json:"location"`
	Timestamp string `json:"timestamp"`
	Notes     string `json:"notes"`
}

func (s *ShipmentContract) shipmentExists(ctx contractapi.TransactionContextInterface, shipmentID string) (bool, error) {
	shipmentJSON, err := ctx.GetStub().GetState(shipmentID)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return shipmentJSON != nil, nil
}

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
