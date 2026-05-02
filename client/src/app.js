/*
 * Express REST API for the shipment tracking chaincode.
 * CSE 540 – Spring B 2026 | Group 1
 */

'use strict';

const express = require('express');
const { Gateway, Wallets } = require('fabric-network');
const path = require('path');
const fs = require('fs');
const { ccpPath } = require('./fabricConfig');
const { addToIPFS, computeSHA256, getIPFSGatewayURL } = require('./ipfsClient');

const app = express();
app.use(express.json());
app.use(express.static(path.join(__dirname, '..', 'public')));

const PORT = process.env.PORT || 3000;
const CHANNEL_NAME = 'shipchannel';
const CHAINCODE_NAME = 'shipment';

async function connectToGateway(userId = 'appUser') {
  const ccp = JSON.parse(fs.readFileSync(ccpPath, 'utf8'));
  const walletPath = path.join(__dirname, '..', 'wallet');
  const wallet = await Wallets.newFileSystemWallet(walletPath);

  const identity = await wallet.get(userId);
  if (!identity) {
    throw new Error(
      `Identity "${userId}" not found in wallet. Run 'npm run enroll-admin' and 'npm run register-user' first.`
    );
  }

  const gateway = new Gateway();
  await gateway.connect(ccp, {
    wallet,
    identity: userId,
    discovery: { enabled: true, asLocalhost: true },
  });

  const network = await gateway.getNetwork(CHANNEL_NAME);
  const contract = network.getContract(CHAINCODE_NAME);

  return { gateway, contract };
}

// Health check
app.get('/api/health', (req, res) => {
  res.json({ status: 'OK', service: 'Shipment Tracking API', timestamp: new Date().toISOString() });
});

// POST /api/shipments — Create a new shipment
app.post('/api/shipments', async (req, res) => {
  try {
    const { shipmentID, origin, destination, participants, offChainData } = req.body;

    if (!shipmentID || !origin || !destination || !participants) {
      return res.status(400).json({ error: 'Missing required fields: shipmentID, origin, destination, participants' });
    }

    const { gateway, contract } = await connectToGateway();

    await contract.submitTransaction(
      'CreateShipment',
      shipmentID,
      origin,
      destination,
      JSON.stringify(participants),
      offChainData || ''
    );

    gateway.disconnect();
    res.status(201).json({ message: `Shipment ${shipmentID} created successfully` });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// GET /api/shipments/:id — Get shipment details
app.get('/api/shipments/:id', async (req, res) => {
  try {
    const { gateway, contract } = await connectToGateway();
    const result = await contract.evaluateTransaction('GetShipment', req.params.id);
    gateway.disconnect();

    res.json(JSON.parse(result.toString()));
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// GET /api/shipments — Get all shipments (paginated if ?pageSize= is provided)
app.get('/api/shipments', async (req, res) => {
  try {
    const { gateway, contract } = await connectToGateway();
    const { pageSize, bookmark } = req.query;

    if (pageSize) {
      const result = await contract.evaluateTransaction(
        'GetShipmentsPaginated',
        String(pageSize),
        bookmark || ''
      );
      gateway.disconnect();
      return res.json(JSON.parse(result.toString()));
    }

    const result = await contract.evaluateTransaction('GetAllShipments');
    gateway.disconnect();
    res.json(JSON.parse(result.toString()));
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// PUT /api/shipments/:id/status — Update shipment status
app.put('/api/shipments/:id/status', async (req, res) => {
  try {
    const { status, location, notes } = req.body;

    if (!status) {
      return res.status(400).json({ error: 'Missing required field: status' });
    }

    const { gateway, contract } = await connectToGateway();
    await contract.submitTransaction(
      'UpdateShipmentStatus',
      req.params.id,
      status,
      location || '',
      notes || ''
    );

    gateway.disconnect();
    res.json({ message: `Shipment ${req.params.id} status updated to '${status}'` });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// PUT /api/shipments/:id/transfer — Transfer custody
app.put('/api/shipments/:id/transfer', async (req, res) => {
  try {
    const { newHolder } = req.body;

    if (!newHolder) {
      return res.status(400).json({ error: 'Missing required field: newHolder' });
    }

    const { gateway, contract } = await connectToGateway();
    await contract.submitTransaction('TransferCustody', req.params.id, newHolder);

    gateway.disconnect();
    res.json({ message: `Custody of ${req.params.id} transferred to ${newHolder}` });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// GET /api/shipments/:id/verify — Verify shipment data integrity
app.get('/api/shipments/:id/verify', async (req, res) => {
  try {
    const { offChainData } = req.query;

    if (!offChainData) {
      return res.status(400).json({ error: 'Missing query parameter: offChainData' });
    }

    const { gateway, contract } = await connectToGateway();
    const result = await contract.evaluateTransaction('VerifyShipment', req.params.id, offChainData);
    gateway.disconnect();

    res.json({ verified: result.toString() === 'true' });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// GET /api/shipments/:id/history — Get shipment event history
app.get('/api/shipments/:id/history', async (req, res) => {
  try {
    const { gateway, contract } = await connectToGateway();
    const result = await contract.evaluateTransaction('GetShipmentHistory', req.params.id);
    gateway.disconnect();

    res.json(JSON.parse(result.toString()));
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// POST /api/shipments/:id/participants — Authorize a participant
app.post('/api/shipments/:id/participants', async (req, res) => {
  try {
    const { participant } = req.body;

    if (!participant) {
      return res.status(400).json({ error: 'Missing required field: participant' });
    }

    const { gateway, contract } = await connectToGateway();
    await contract.submitTransaction('AuthorizeParticipant', req.params.id, participant);

    gateway.disconnect();
    res.json({ message: `Participant ${participant} authorized for shipment ${req.params.id}` });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// DELETE /api/shipments/:id/participants/:participant — Revoke a participant
app.delete('/api/shipments/:id/participants/:participant', async (req, res) => {
  try {
    const { gateway, contract } = await connectToGateway();
    await contract.submitTransaction('RevokeParticipant', req.params.id, req.params.participant);

    gateway.disconnect();
    res.json({ message: `Participant ${req.params.participant} revoked from shipment ${req.params.id}` });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// POST /api/shipments/:id/devices — Register an IoT device for a shipment
app.post('/api/shipments/:id/devices', async (req, res) => {
  try {
    const { deviceID, deviceType } = req.body;

    if (!deviceID || !deviceType) {
      return res.status(400).json({ error: 'Missing required fields: deviceID, deviceType' });
    }

    const { gateway, contract } = await connectToGateway();
    await contract.submitTransaction('RegisterDevice', req.params.id, deviceID, deviceType);

    gateway.disconnect();
    res.json({ message: `Device ${deviceID} (${deviceType}) registered for shipment ${req.params.id}` });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// POST /api/shipments/:id/telemetry — Record a simulated IoT sensor reading
// Requires the device to be pre-registered via POST /api/shipments/:id/devices
app.post('/api/shipments/:id/telemetry', async (req, res) => {
  try {
    const { deviceID, sensorType, value, unit, location } = req.body;

    if (!deviceID || !sensorType || value === undefined) {
      return res.status(400).json({ error: 'Missing required fields: deviceID, sensorType, value' });
    }

    const { gateway, contract } = await connectToGateway();
    await contract.submitTransaction(
      'RecordTelemetry',
      req.params.id,
      deviceID,
      sensorType,
      String(value),
      unit || '',
      location || ''
    );

    gateway.disconnect();
    res.json({ message: `Telemetry recorded for shipment ${req.params.id}: ${sensorType}=${value}${unit || ''}` });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// POST /api/shipments/:id/documents — Upload a document to IPFS and record CID + hash on-chain
app.post('/api/shipments/:id/documents', async (req, res) => {
  try {
    const { content, docType, description } = req.body;

    if (!content || !docType) {
      return res.status(400).json({ error: 'Missing required fields: content, docType' });
    }

    const docHash = computeSHA256(content);
    const { cid } = await addToIPFS(content);
    const gatewayURL = getIPFSGatewayURL(cid);

    const { gateway, contract } = await connectToGateway();
    await contract.submitTransaction(
      'RecordDocument',
      req.params.id,
      cid,
      docHash,
      docType,
      description || ''
    );

    gateway.disconnect();
    res.status(201).json({
      message: `Document recorded for shipment ${req.params.id}`,
      cid,
      hash: docHash,
      gateway: gatewayURL
    });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

app.listen(PORT, () => {
  console.log(`\n========================================`);
  console.log(`  Shipment Tracking API`);
  console.log(`  Listening on port ${PORT}`);
  console.log(`  Channel: ${CHANNEL_NAME}`);
  console.log(`  Chaincode: ${CHAINCODE_NAME}`);
  console.log(`========================================\n`);
});

module.exports = app;
