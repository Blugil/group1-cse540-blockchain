/*
 * Runs through the main shipment flows end to end:
 * create → update status → transfer custody → verify → check history.
 */

'use strict';

const { Gateway, Wallets } = require('fabric-network');
const path = require('path');
const fs = require('fs');
const { ccpPath } = require('./fabricConfig');

const CHANNEL_NAME = 'shipchannel';
const CHAINCODE_NAME = 'shipment';

async function main() {
  let gateway;

  try {
    const ccp = JSON.parse(fs.readFileSync(ccpPath, 'utf8'));
    const walletPath = path.join(__dirname, '..', 'wallet');
    const wallet = await Wallets.newFileSystemWallet(walletPath);

    const identity = await wallet.get('appUser');
    if (!identity) {
      console.log('ERROR: appUser identity not found. Run enrollment scripts first.');
      return;
    }

    // Connect to gateway
    gateway = new Gateway();
    await gateway.connect(ccp, {
      wallet,
      identity: 'appUser',
      discovery: { enabled: true, asLocalhost: true },
    });

    const network = await gateway.getNetwork(CHANNEL_NAME);
    const contract = network.getContract(CHAINCODE_NAME);

    console.log('\n========================================');
    console.log('  Shipment Tracking — Test Workflow');
    console.log('========================================\n');

    // ---- Step 1: Query sample shipment ----
    console.log('--- Step 1: Query Sample Shipment (SHIP-001) ---');
    let result = await contract.evaluateTransaction('GetShipment', 'SHIP-001');
    console.log(`Result: ${prettyJSON(result)}\n`);

    // ---- Step 2: Create a new shipment ----
    console.log('--- Step 2: Create New Shipment (SHIP-TEST-001) ---');
    const participants = JSON.stringify([
      'ManufacturerMSP',
      'TransporterMSP',
      'WarehouseMSP',
      'RetailerMSP',
      'RecipientMSP',
    ]);
    const offChainData = 'weight:25kg,volume:1cbm,count:50,contents:electronics';
    await contract.submitTransaction(
      'CreateShipment',
      'SHIP-TEST-001',
      'Factory-San-Jose-CA',
      'Store-New-York-NY',
      participants,
      offChainData
    );
    console.log('Shipment SHIP-TEST-001 created.\n');

    // ---- Step 3: Query the new shipment ----
    console.log('--- Step 3: Query New Shipment ---');
    result = await contract.evaluateTransaction('GetShipment', 'SHIP-TEST-001');
    console.log(`Result: ${prettyJSON(result)}\n`);

    // ---- Step 4: Update status ----
    console.log('--- Step 4: Update Status to InTransit ---');
    await contract.submitTransaction(
      'UpdateShipmentStatus',
      'SHIP-TEST-001',
      'InTransit',
      'Loading-Dock-San-Jose',
      'Package loaded onto truck'
    );
    console.log('Status updated.\n');

    // ---- Step 5: Transfer custody ----
    console.log('--- Step 5: Transfer Custody to TransporterMSP ---');
    await contract.submitTransaction('TransferCustody', 'SHIP-TEST-001', 'TransporterMSP');
    console.log('Custody transferred.\n');

    // ---- Step 6: Query updated shipment ----
    console.log('--- Step 6: Query Updated Shipment ---');
    result = await contract.evaluateTransaction('GetShipment', 'SHIP-TEST-001');
    console.log(`Result: ${prettyJSON(result)}\n`);

    // ---- Step 7: Verify shipment integrity ----
    console.log('--- Step 7: Verify Shipment Data Integrity ---');
    result = await contract.evaluateTransaction('VerifyShipment', 'SHIP-TEST-001', offChainData);
    console.log(`Verified: ${result.toString()}\n`);

    // ---- Step 8: Get shipment history ----
    console.log('--- Step 8: Get Shipment History ---');
    result = await contract.evaluateTransaction('GetShipmentHistory', 'SHIP-TEST-001');
    console.log(`History: ${prettyJSON(result)}\n`);

    // ---- Step 9: Get all shipments ----
    console.log('--- Step 9: Get All Shipments ---');
    result = await contract.evaluateTransaction('GetAllShipments');
    console.log(`All Shipments: ${prettyJSON(result)}\n`);

    console.log('========================================');
    console.log('  Test Workflow Complete!');
    console.log('========================================\n');
  } catch (error) {
    console.error(`Test workflow failed: ${error}`);
    process.exit(1);
  } finally {
    if (gateway) {
      gateway.disconnect();
    }
  }
}

function prettyJSON(buffer) {
  try {
    return JSON.stringify(JSON.parse(buffer.toString()), null, 2);
  } catch {
    return buffer.toString();
  }
}

main();
