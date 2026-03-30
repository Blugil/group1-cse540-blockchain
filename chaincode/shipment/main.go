// This is a planned out set of functions that could likely be necessary or valuable for the implementation of this prject discussed as a group. Many of these functions could change depending on discovered uses or unnecessary functions may be discarded.
// @authors: Dominick Agnello, Ritish Abrol, Vatsal Patel, Shashikant Nanda, Anushree Bhure

package main

import "github.com/hyperledger/fabric-contract-api-go/contractapi"

type ShipmentContract struct {
	contractapi.Contract
}

// some necessary information about the shipment, many of these are self explanatory
type Shipment struct {
	ShipmentID    string   `json:"shipmentID"`
	Status        string   `json:"status"`
	Origin        string   `json:"origin"`
	Destination   string   `json:"destination"`
	CurrentHolder string   `json:"currentHolder"`
	Participants  []string `json:"participants"`
	Timestamp     string   `json:"timestamp"`
}

// related to the status of the shipment throughout the blockchain process.
type ShipmentEvent struct {
	EventType string `json:"eventType"`
	Actor     string `json:"actor"`
	Location  string `json:"location"`
	Timestamp string `json:"timestamp"`
	Notes     string `json:"notes"`
}

// This is a simple intializer function, only runs once when the chaincode is deployed and will handle ledger initialization, probably unnecessary but its a safe bet right now
func (s *ShipmentContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	//TODO 
}

// this function registers a new shipment on the ledger with the origin, destinations,and permitted interaction points. It's possible this gets changed, but the main idea is that only an exact set of entities are allowed to interact with a shipment and deviation isn't permitted lest it break the chain
func (s *ShipmentContract) CreateShipment(ctx contractapi.TransactionContextInterface, shipmentID string, origin string, destination string, participants []string) error {
	//TODO
}

// returns info about the shipment called by ID
func (s *ShipmentContract) GetShipment(ctx contractapi.TransactionContextInterface, shipmentID string) (*Shipment, error) {
	//TODO
}

// general shipment status updates are broadcast and attached to the contract, examples could be "package passed off", "delivered", etc. Seemed useful in planning but we might ditch it in the actual creation of the blockchain program.
func (s *ShipmentContract) UpdateShipmentStatus(ctx contractapi.TransactionContextInterface, shipmentID string, status string, location string, notes string) error {
	//TODO
}

// changes the entity currently holding a shipment, this is effectively the "transaction" so to speak, in coin terms.
func (s *ShipmentContract) TransferCustody(ctx contractapi.TransactionContextInterface, shipmentID string, newHolder string) error {
	//TODO
}

// validates the state of a shipment against the ledger
func (s *ShipmentContract) VerifyShipment(ctx contractapi.TransactionContextInterface, shipmentID string) (bool, error) {
	//TODO
}

// retrieves the entire set of historty pertaining to a given shipment. likely to be unnecessary but included in planning to have potential uses
func (s *ShipmentContract) GetShipmentHistory(ctx contractapi.TransactionContextInterface, shipmentID string) ([]*ShipmentEvent, error) {
	//TODO
}

// in the event that a new entity needs contol over a system, given confirmation from all involved parties (recipient, shipper, current entity, and new entity) permission can be given.
func (s *ShipmentContract) AuthorizeParticipant(ctx contractapi.TransactionContextInterface, shipmentID string, participant string) error {
	//TODO
}

// the opposite of the above, revokes under the persmissions of the shipper, recipient, and current shipment entity.
func (s *ShipmentContract) RevokeParticipant(ctx contractapi.TransactionContextInterface, shipmentID string, participant string) error {
	//TODO
}


func main() {
	cc, _ := contractapi.NewChaincode(&ShipmentContract{})
	cc.Start()
}
