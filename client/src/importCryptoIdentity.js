'use strict';
/*
 * Imports the cryptogen-generated Admin and User1 identities from the Fabric test-network
 * into the local wallet. Run this once after the network is up before starting the API.
 */

const { Wallets } = require('fabric-network');
const fs = require('fs');
const path = require('path');
const { org1Path } = require('./fabricConfig');

async function main() {
  const walletPath = path.join(__dirname, '..', 'wallet');
  const wallet = await Wallets.newFileSystemWallet(walletPath);
  console.log(`Wallet path: ${walletPath}`);

  const adminKeyDir  = path.join(org1Path, 'users', 'Admin@org1.example.com', 'msp', 'keystore');
  const adminCertDir = path.join(org1Path, 'users', 'Admin@org1.example.com', 'msp', 'signcerts');
  const adminKey  = fs.readFileSync(path.join(adminKeyDir,  fs.readdirSync(adminKeyDir)[0]),  'utf8');
  const adminCert = fs.readFileSync(path.join(adminCertDir, fs.readdirSync(adminCertDir)[0]), 'utf8');

  await wallet.put('admin', {
    credentials: { certificate: adminCert, privateKey: adminKey },
    mspId: 'Org1MSP',
    type: 'X.509',
  });
  console.log('Imported Admin@org1.example.com as "admin"');

  const userKeyDir  = path.join(org1Path, 'users', 'User1@org1.example.com', 'msp', 'keystore');
  const userCertDir = path.join(org1Path, 'users', 'User1@org1.example.com', 'msp', 'signcerts');
  const userKey  = fs.readFileSync(path.join(userKeyDir,  fs.readdirSync(userKeyDir)[0]),  'utf8');
  const userCert = fs.readFileSync(path.join(userCertDir, fs.readdirSync(userCertDir)[0]), 'utf8');

  await wallet.put('appUser', {
    credentials: { certificate: userCert, privateKey: userKey },
    mspId: 'Org1MSP',
    type: 'X.509',
  });
  console.log('Imported User1@org1.example.com as "appUser"');
}

main().catch(e => { console.error(e); process.exit(1); });
