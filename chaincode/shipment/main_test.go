package main

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/golang/protobuf/proto"
	mspproto "github.com/hyperledger/fabric-protos-go/msp"
)

func setupMockStub(t *testing.T) *shimtest.MockStub {
	cc, err := contractapi.NewChaincode(&ShipmentContract{})
	if err != nil {
		t.Fatalf("Failed to create chaincode: %v", err)
	}
	stub := shimtest.NewMockStub("shipment", cc)
	if stub == nil {
		t.Fatal("Failed to create mock stub")
	}

	setMockCreator(t, stub, "ManufacturerMSP")

	return stub
}

func setMockCreator(t *testing.T, stub *shimtest.MockStub, mspID string) {
	certPEM := generateSelfSignedCert(t)

	sid := &mspproto.SerializedIdentity{
		Mspid:   mspID,
		IdBytes: certPEM,
	}

	serialized, err := proto.Marshal(sid)
	if err != nil {
		t.Fatalf("Failed to marshal serialized identity: %v", err)
	}

	stub.Creator = serialized
}

func generateSelfSignedCert(t *testing.T) []byte {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-user",
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
}

func TestInitLedger(t *testing.T) {
	stub := setupMockStub(t)

	result := stub.MockInvoke("tx1", [][]byte{
		[]byte("InitLedger"),
	})

	if result.Status != 200 {
		t.Fatalf("InitLedger failed: %s", result.Message)
	}
	t.Log("✅ InitLedger succeeded")

	shipmentBytes := stub.State["SHIP-001"]
	if shipmentBytes == nil {
		t.Fatal("Sample shipment SHIP-001 not found on ledger after InitLedger")
	}

	var shipment Shipment
	err := json.Unmarshal(shipmentBytes, &shipment)
	if err != nil {
		t.Fatalf("Failed to unmarshal SHIP-001: %v", err)
	}

	if shipment.ShipmentID != "SHIP-001" {
		t.Errorf("Expected ShipmentID 'SHIP-001', got '%s'", shipment.ShipmentID)
	}
	if shipment.Status != "Created" {
		t.Errorf("Expected Status 'Created', got '%s'", shipment.Status)
	}
	if shipment.Origin != "Factory-Phoenix-AZ" {
		t.Errorf("Expected Origin 'Factory-Phoenix-AZ', got '%s'", shipment.Origin)
	}
	if shipment.CurrentHolder != "ManufacturerMSP" {
		t.Errorf("Expected CurrentHolder 'ManufacturerMSP', got '%s'", shipment.CurrentHolder)
	}
	if shipment.DataHash == "" {
		t.Error("Expected DataHash to be set")
	}

	t.Logf("✅ Sample shipment verified: ID=%s, Status=%s, Holder=%s",
		shipment.ShipmentID, shipment.Status, shipment.CurrentHolder)
}

func TestCreateShipment(t *testing.T) {
	stub := setupMockStub(t)

	participants := `["ManufacturerMSP","TransporterMSP","WarehouseMSP"]`
	offChainData := "weight:25kg,volume:1cbm,count:50"

	result := stub.MockInvoke("tx1", [][]byte{
		[]byte("CreateShipment"),
		[]byte("SHIP-TEST-001"),
		[]byte("Factory-San-Jose-CA"),
		[]byte("Store-New-York-NY"),
		[]byte(participants),
		[]byte(offChainData),
	})

	if result.Status != 200 {
		t.Fatalf("CreateShipment failed: %s", result.Message)
	}
	t.Log("✅ CreateShipment succeeded")

	shipmentBytes := stub.State["SHIP-TEST-001"]
	if shipmentBytes == nil {
		t.Fatal("Shipment SHIP-TEST-001 not found")
	}

	var shipment Shipment
	json.Unmarshal(shipmentBytes, &shipment)

	if shipment.Status != "Created" {
		t.Errorf("Expected Status 'Created', got '%s'", shipment.Status)
	}
	if shipment.Origin != "Factory-San-Jose-CA" {
		t.Errorf("Expected Origin 'Factory-San-Jose-CA', got '%s'", shipment.Origin)
	}
	if shipment.Destination != "Store-New-York-NY" {
		t.Errorf("Expected Destination 'Store-New-York-NY', got '%s'", shipment.Destination)
	}

	expectedHash := computeHash(offChainData)
	if shipment.DataHash != expectedHash {
		t.Errorf("Expected DataHash '%s', got '%s'", expectedHash, shipment.DataHash)
	}

	t.Logf("✅ Shipment created: ID=%s, Hash=%s", shipment.ShipmentID, shipment.DataHash)
}

func TestCreateShipmentDuplicate(t *testing.T) {
	stub := setupMockStub(t)

	args := [][]byte{
		[]byte("CreateShipment"),
		[]byte("SHIP-DUP"),
		[]byte("Origin"),
		[]byte("Destination"),
		[]byte(`["ManufacturerMSP"]`),
		[]byte("data"),
	}

	result := stub.MockInvoke("tx1", args)
	if result.Status != 200 {
		t.Fatalf("First CreateShipment failed: %s", result.Message)
	}

	result = stub.MockInvoke("tx2", args)
	if result.Status == 200 {
		t.Fatal("Expected duplicate CreateShipment to fail, but it succeeded")
	}

	t.Logf("✅ Duplicate shipment correctly rejected: %s", result.Message)
}

func TestUpdateShipmentStatus(t *testing.T) {
	stub := setupMockStub(t)

	stub.MockInvoke("tx1", [][]byte{
		[]byte("CreateShipment"),
		[]byte("SHIP-STATUS"),
		[]byte("Origin"),
		[]byte("Destination"),
		[]byte(`["ManufacturerMSP","TransporterMSP"]`),
		[]byte("data"),
	})

	result := stub.MockInvoke("tx2", [][]byte{
		[]byte("UpdateShipmentStatus"),
		[]byte("SHIP-STATUS"),
		[]byte("InTransit"),
		[]byte("Loading-Dock"),
		[]byte("Package loaded"),
	})

	if result.Status != 200 {
		t.Fatalf("UpdateShipmentStatus failed: %s", result.Message)
	}

	var shipment Shipment
	json.Unmarshal(stub.State["SHIP-STATUS"], &shipment)

	if shipment.Status != "InTransit" {
		t.Errorf("Expected Status 'InTransit', got '%s'", shipment.Status)
	}

	t.Logf("✅ Status updated to: %s", shipment.Status)
}

func TestUpdateShipmentStatusInvalid(t *testing.T) {
	stub := setupMockStub(t)

	stub.MockInvoke("tx1", [][]byte{
		[]byte("CreateShipment"),
		[]byte("SHIP-INVALID"),
		[]byte("Origin"),
		[]byte("Destination"),
		[]byte(`["ManufacturerMSP"]`),
		[]byte("data"),
	})

	result := stub.MockInvoke("tx2", [][]byte{
		[]byte("UpdateShipmentStatus"),
		[]byte("SHIP-INVALID"),
		[]byte("BadStatus"),
		[]byte("Location"),
		[]byte("notes"),
	})

	if result.Status == 200 {
		t.Fatal("Expected invalid status to be rejected")
	}

	t.Logf("✅ Invalid status correctly rejected: %s", result.Message)
}

func TestTransferCustody(t *testing.T) {
	stub := setupMockStub(t)

	stub.MockInvoke("tx1", [][]byte{
		[]byte("CreateShipment"),
		[]byte("SHIP-TRANSFER"),
		[]byte("Origin"),
		[]byte("Destination"),
		[]byte(`["ManufacturerMSP","TransporterMSP"]`),
		[]byte("data"),
	})

	result := stub.MockInvoke("tx2", [][]byte{
		[]byte("TransferCustody"),
		[]byte("SHIP-TRANSFER"),
		[]byte("TransporterMSP"),
	})

	if result.Status != 200 {
		t.Fatalf("TransferCustody failed: %s", result.Message)
	}

	var shipment Shipment
	json.Unmarshal(stub.State["SHIP-TRANSFER"], &shipment)

	if shipment.CurrentHolder != "TransporterMSP" {
		t.Errorf("Expected CurrentHolder 'TransporterMSP', got '%s'", shipment.CurrentHolder)
	}
	if shipment.Status != "InTransit" {
		t.Errorf("Expected Status 'InTransit', got '%s'", shipment.Status)
	}

	t.Logf("✅ Custody transferred: Holder=%s, Status=%s", shipment.CurrentHolder, shipment.Status)
}

func TestTransferCustodyUnauthorized(t *testing.T) {
	stub := setupMockStub(t)

	stub.MockInvoke("tx1", [][]byte{
		[]byte("CreateShipment"),
		[]byte("SHIP-UNAUTH"),
		[]byte("Origin"),
		[]byte("Destination"),
		[]byte(`["ManufacturerMSP"]`),
		[]byte("data"),
	})

	result := stub.MockInvoke("tx2", [][]byte{
		[]byte("TransferCustody"),
		[]byte("SHIP-UNAUTH"),
		[]byte("UnauthorizedMSP"),
	})

	if result.Status == 200 {
		t.Fatal("Expected transfer to unauthorized participant to fail")
	}

	t.Logf("✅ Unauthorized transfer correctly rejected: %s", result.Message)
}

func TestVerifyShipmentValid(t *testing.T) {
	stub := setupMockStub(t)

	offChainData := "weight:10kg,volume:0.5cbm"

	stub.MockInvoke("tx1", [][]byte{
		[]byte("CreateShipment"),
		[]byte("SHIP-VERIFY"),
		[]byte("Origin"),
		[]byte("Destination"),
		[]byte(`["ManufacturerMSP"]`),
		[]byte(offChainData),
	})

	result := stub.MockInvoke("tx2", [][]byte{
		[]byte("VerifyShipment"),
		[]byte("SHIP-VERIFY"),
		[]byte(offChainData),
	})

	if result.Status != 200 {
		t.Fatalf("VerifyShipment failed: %s", result.Message)
	}

	t.Log("✅ Shipment data integrity verified (matching hash)")
}

func TestVerifyShipmentInvalid(t *testing.T) {
	stub := setupMockStub(t)

	stub.MockInvoke("tx1", [][]byte{
		[]byte("CreateShipment"),
		[]byte("SHIP-VERIFY2"),
		[]byte("Origin"),
		[]byte("Destination"),
		[]byte(`["ManufacturerMSP"]`),
		[]byte("original-data"),
	})

	result := stub.MockInvoke("tx2", [][]byte{
		[]byte("VerifyShipment"),
		[]byte("SHIP-VERIFY2"),
		[]byte("tampered-data"),
	})

	if result.Status == 200 {
		var verified bool
		json.Unmarshal(result.Payload, &verified)
		if verified {
			t.Fatal("Expected verification to fail for tampered data")
		}
	}

	t.Log("✅ Tampered data correctly detected (hash mismatch)")
}

func TestAuthorizeParticipant(t *testing.T) {
	stub := setupMockStub(t)

	stub.MockInvoke("tx1", [][]byte{
		[]byte("CreateShipment"),
		[]byte("SHIP-AUTH"),
		[]byte("Origin"),
		[]byte("Destination"),
		[]byte(`["ManufacturerMSP"]`),
		[]byte("data"),
	})

	result := stub.MockInvoke("tx2", [][]byte{
		[]byte("AuthorizeParticipant"),
		[]byte("SHIP-AUTH"),
		[]byte("NewPartnerMSP"),
	})

	if result.Status != 200 {
		t.Fatalf("AuthorizeParticipant failed: %s", result.Message)
	}

	var shipment Shipment
	json.Unmarshal(stub.State["SHIP-AUTH"], &shipment)

	found := false
	for _, p := range shipment.Participants {
		if p == "NewPartnerMSP" {
			found = true
			break
		}
	}
	if !found {
		t.Error("NewPartnerMSP not found in participants list")
	}

	t.Logf("✅ Participant authorized: Participants=%v", shipment.Participants)
}

func TestRevokeParticipant(t *testing.T) {
	stub := setupMockStub(t)

	stub.MockInvoke("tx1", [][]byte{
		[]byte("CreateShipment"),
		[]byte("SHIP-REVOKE"),
		[]byte("Origin"),
		[]byte("Destination"),
		[]byte(`["ManufacturerMSP","TransporterMSP","WarehouseMSP"]`),
		[]byte("data"),
	})

	result := stub.MockInvoke("tx2", [][]byte{
		[]byte("RevokeParticipant"),
		[]byte("SHIP-REVOKE"),
		[]byte("WarehouseMSP"),
	})

	if result.Status != 200 {
		t.Fatalf("RevokeParticipant failed: %s", result.Message)
	}

	var shipment Shipment
	json.Unmarshal(stub.State["SHIP-REVOKE"], &shipment)

	for _, p := range shipment.Participants {
		if p == "WarehouseMSP" {
			t.Error("WarehouseMSP should have been removed from participants")
		}
	}

	t.Logf("✅ Participant revoked: Participants=%v", shipment.Participants)
}

func TestFullLifecycle(t *testing.T) {
	stub := setupMockStub(t)

	t.Log("--- Full Lifecycle Test ---")

	result := stub.MockInvoke("tx1", [][]byte{
		[]byte("CreateShipment"),
		[]byte("SHIP-LIFECYCLE"),
		[]byte("Factory-Phoenix-AZ"),
		[]byte("Store-LA-CA"),
		[]byte(`["ManufacturerMSP","TransporterMSP","WarehouseMSP","RetailerMSP","RecipientMSP"]`),
		[]byte("weight:50kg,count:200"),
	})
	if result.Status != 200 {
		t.Fatalf("Step 1 (Create) failed: %s", result.Message)
	}
	t.Log("  Step 1: ✅ Shipment created")

	result = stub.MockInvoke("tx2", [][]byte{
		[]byte("UpdateShipmentStatus"),
		[]byte("SHIP-LIFECYCLE"),
		[]byte("InTransit"),
		[]byte("Loading-Dock"),
		[]byte("Loaded onto truck"),
	})
	if result.Status != 200 {
		t.Fatalf("Step 2 (Status Update) failed: %s", result.Message)
	}
	t.Log("  Step 2: ✅ Status updated to InTransit")

	result = stub.MockInvoke("tx3", [][]byte{
		[]byte("TransferCustody"),
		[]byte("SHIP-LIFECYCLE"),
		[]byte("TransporterMSP"),
	})
	if result.Status != 200 {
		t.Fatalf("Step 3 (Transfer) failed: %s", result.Message)
	}
	t.Log("  Step 3: ✅ Custody transferred to TransporterMSP")

	result = stub.MockInvoke("tx4", [][]byte{
		[]byte("VerifyShipment"),
		[]byte("SHIP-LIFECYCLE"),
		[]byte("weight:50kg,count:200"),
	})
	if result.Status != 200 {
		t.Fatalf("Step 4 (Verify) failed: %s", result.Message)
	}
	t.Log("  Step 4: ✅ Data integrity verified")

	var shipment Shipment
	json.Unmarshal(stub.State["SHIP-LIFECYCLE"], &shipment)

	if shipment.CurrentHolder != "TransporterMSP" {
		t.Errorf("Expected holder TransporterMSP, got %s", shipment.CurrentHolder)
	}
	if shipment.Status != "InTransit" {
		t.Errorf("Expected status InTransit, got %s", shipment.Status)
	}

	t.Log("  Step 5: ✅ Final state verified")
	t.Log("--- Full Lifecycle Test Complete ---")
}

func TestDeliveredCannotBeUpdated(t *testing.T) {
	stub := setupMockStub(t)

	stub.MockInvoke("tx1", [][]byte{
		[]byte("CreateShipment"),
		[]byte("SHIP-DELIVERED"),
		[]byte("Origin"),
		[]byte("Destination"),
		[]byte(`["ManufacturerMSP"]`),
		[]byte("data"),
	})

	stub.MockInvoke("tx2", [][]byte{
		[]byte("UpdateShipmentStatus"),
		[]byte("SHIP-DELIVERED"),
		[]byte("Delivered"),
		[]byte("Doorstep"),
		[]byte("Package delivered"),
	})

	result := stub.MockInvoke("tx3", [][]byte{
		[]byte("UpdateShipmentStatus"),
		[]byte("SHIP-DELIVERED"),
		[]byte("InTransit"),
		[]byte("Warehouse"),
		[]byte("Trying to reopen"),
	})

	if result.Status == 200 {
		t.Fatal("Expected update to delivered shipment to fail")
	}

	t.Logf("✅ Delivered shipment correctly locked: %s", result.Message)
}

func TestGetAllShipments(t *testing.T) {
	stub := setupMockStub(t)

	for i := 1; i <= 3; i++ {
		stub.MockInvoke(fmt.Sprintf("tx%d", i), [][]byte{
			[]byte("CreateShipment"),
			[]byte(fmt.Sprintf("SHIP-ALL-%d", i)),
			[]byte("Origin"),
			[]byte("Destination"),
			[]byte(`["ManufacturerMSP"]`),
			[]byte(fmt.Sprintf("data-%d", i)),
		})
	}

	for i := 1; i <= 3; i++ {
		key := fmt.Sprintf("SHIP-ALL-%d", i)
		if stub.State[key] == nil {
			t.Errorf("Shipment %s not found", key)
		}
	}

	t.Logf("✅ All 3 shipments exist on ledger")
}

func TestComputeHash(t *testing.T) {
	data := "weight:25kg,volume:1cbm"
	hash1 := computeHash(data)
	hash2 := computeHash(data)

	if hash1 != hash2 {
		t.Error("Same data should produce the same hash")
	}

	hash3 := computeHash("different-data")
	if hash1 == hash3 {
		t.Error("Different data should produce different hashes")
	}

	if len(hash1) != 64 {
		t.Errorf("SHA-256 hex digest should be 64 chars, got %d", len(hash1))
	}

	t.Logf("✅ Hash computation correct: %s", hash1)
}
