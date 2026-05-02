import json
import os

script_dir = os.path.dirname(os.path.abspath(__file__))
ndir = os.path.join(script_dir, "network")

with open(os.path.join(ndir, "organizations/peerOrganizations/manufacturer.shipment.com/peers/peer0.manufacturer.shipment.com/tls/ca.crt")) as f:
    mfr_tls = f.read().strip()

with open(os.path.join(ndir, "organizations/ordererOrganizations/shipment.com/orderers/orderer.shipment.com/tls/ca.crt")) as f:
    ord_tls = f.read().strip()

with open(os.path.join(ndir, "organizations/peerOrganizations/manufacturer.shipment.com/ca/ca.manufacturer.shipment.com-cert.pem")) as f:
    mfr_ca = f.read().strip()

conn = {
  "name": "shipment-network",
  "version": "1.0.0",
  "client": {
    "organization": "ManufacturerMSP",
    "connection": {"timeout": {"peer": {"endorser": "300"}, "orderer": "300"}}
  },
  "organizations": {
    "ManufacturerMSP": {
      "mspid": "ManufacturerMSP",
      "peers": ["peer0.manufacturer.shipment.com"],
      "certificateAuthorities": ["ca.manufacturer.shipment.com"]
    }
  },
  "peers": {
    "peer0.manufacturer.shipment.com": {
      "url": "grpcs://localhost:7051",
      "tlsCACerts": {"pem": mfr_tls},
      "grpcOptions": {
        "ssl-target-name-override": "peer0.manufacturer.shipment.com",
        "hostnameOverride": "peer0.manufacturer.shipment.com"
      }
    }
  },
  "orderers": {
    "orderer.shipment.com": {
      "url": "grpcs://localhost:7050",
      "tlsCACerts": {"pem": ord_tls},
      "grpcOptions": {"ssl-target-name-override": "orderer.shipment.com"}
    }
  },
  "certificateAuthorities": {
    "ca.manufacturer.shipment.com": {
      "url": "https://localhost:7054",
      "caName": "ca-manufacturer",
      "tlsCACerts": {"pem": [mfr_ca]},
      "httpOptions": {"verify": False}
    }
  }
}

out = os.path.join(script_dir, "client", "connection-manufacturer.json")
with open(out, "w") as f:
    json.dump(conn, f, indent=2)

print("Written successfully:", out)
